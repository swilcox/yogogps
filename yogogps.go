package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/swilcox/go-gpsd"
)

var templates = template.Must(template.ParseFiles("templates/home.html"))

func renderTemplate(w http.ResponseWriter, tmpl string) {
	err := templates.ExecuteTemplate(w, tmpl+".html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type UnsubscribeFunc func() error

type Subscriber interface {
	Subscribe(c chan []byte) (UnsubscribeFunc, error)
}

func handleSSE(s Subscriber) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Subscribe
		c := make(chan []byte)
		unsubscribeFn, err := s.Subscribe(c)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
	Looping:
		for {
			select {
			case <-r.Context().Done():
				if err := unsubscribeFn(); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				break Looping
			default:
				b := <-c
				fmt.Fprintf(w, "data: %s\n\n", b)
				w.(http.Flusher).Flush()
			}
		}
	}
}

type Notifier interface {
	Notify(b []byte) error
}

type NotificationCenter struct {
	subscribers   map[chan []byte]struct{}
	subscribersMu *sync.Mutex
	LastKnownTPV  *AugmentedTPV
	LastKnownSKY  *gpsd.SKYReport
}

func NewNotificationCenter() *NotificationCenter {
	return &NotificationCenter{
		subscribers:   map[chan []byte]struct{}{},
		subscribersMu: &sync.Mutex{},
	}
}

func (nc *NotificationCenter) Subscribe(c chan []byte) (UnsubscribeFunc, error) {
	nc.subscribersMu.Lock()
	nc.subscribers[c] = struct{}{}
	nc.subscribersMu.Unlock()

	unsubscribeFn := func() error {
		nc.subscribersMu.Lock()
		delete(nc.subscribers, c)
		nc.subscribersMu.Unlock()
		return nil
	}
	return unsubscribeFn, nil
}

func (nc *NotificationCenter) Notify(b []byte) error {
	nc.subscribersMu.Lock()
	defer nc.subscribersMu.Unlock()

	var tpv AugmentedTPV
	if err := json.Unmarshal(b, &tpv); err == nil {
		if tpv.Class == "TPV" {
			nc.LastKnownTPV = &tpv
		}
	}

	var sky gpsd.SKYReport
	if err := json.Unmarshal(b, &sky); err == nil {
		if sky.Class == "SKY" {
			nc.LastKnownSKY = &sky
		}
	}

	for c := range nc.subscribers {
		select {
		case c <- b:
		default:
		}
	}
	return nil
}

func home(w http.ResponseWriter, req *http.Request) {
	renderTemplate(w, "home")
}

type Metrics struct {
	NumClients     int
	GPSGridSquare  string
	SatelliteCount int
	GPSLat         float64
	GPSLon         float64
}

func (m *Metrics) ToPromString() string {
	return fmt.Sprintf(
		"yogogps_num_clients %d\n"+
			"yogogps_gps_satellites_count %d\n"+
			"yogogps_gps_lat %f\n"+
			"yogogps_gps_lon %f\n"+
			"yogogps_gps_gridsquare %s\n",
		m.NumClients, m.SatelliteCount, m.GPSLat, m.GPSLon, m.GPSGridSquare,
	)
}

func (m *Metrics) ToPromByteArray() []byte {
	return []byte(m.ToPromString())
}

func handleMetrics(nc *NotificationCenter) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		nc.subscribersMu.Lock()
		metrics := &Metrics{
			NumClients:     len(nc.subscribers),
			SatelliteCount: len(nc.LastKnownSKY.Satellites),
			GPSGridSquare:  nc.LastKnownTPV.GridSquare,
			GPSLat:         nc.LastKnownTPV.Lat,
			GPSLon:         nc.LastKnownTPV.Lon,
		}
		nc.subscribersMu.Unlock()
		w.Write(metrics.ToPromByteArray())
	}
}

func ComputeGridSquare(lat float64, lon float64) string {
	gridSquare := ""
	letters := [26]string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
	lon += 180
	lat += 90
	gridSquare += letters[int(math.Floor(lon/20.0))]
	gridSquare += letters[int(math.Floor(lat/10.0))]
	gridSquare += fmt.Sprint(int(math.Floor(lon/2.0)) % 10)
	gridSquare += fmt.Sprint(int(math.Floor(lat)) % 10)
	gridSquare += strings.ToLower(letters[int(math.Floor(math.Mod(lon, 2.0)*12.0))])
	gridSquare += strings.ToLower(letters[int(math.Floor(math.Mod(lat, 1.0)*24.0))])
	return gridSquare
}

type AugmentedTPV struct {
	gpsd.TPVReport
	GridSquare string
}

func main() {
	// note: SSE code from https://gist.github.com/rikonor/e53a33c27ed64861c91a095a59f0aa44
	nc := NewNotificationCenter()
	// gpsd handling
	var gps *gpsd.Session
	var err error
	var gpsd_server = os.Getenv("GPSD_SERVER")
	if gpsd_server == "" {
		gpsd_server = gpsd.DefaultAddress
	}
	if gps, err = gpsd.Dial(gpsd_server); err != nil {
		panic(fmt.Sprintf("Failed to connect to GPSD: %s", err))
	}
	gps.AddFilter("TPV", func(r interface{}) {
		tpv := r.(*gpsd.TPVReport)
		aTPV := AugmentedTPV{*tpv, ComputeGridSquare(tpv.Lat, tpv.Lon)}
		var tpvJSON []byte
		tpvJSON, err = json.Marshal(aTPV)

		if err != nil {
			fmt.Println(err)
		}
		nc.Notify(tpvJSON)
	})
	skyfilter := func(r interface{}) {
		sky := r.(*gpsd.SKYReport)
		var skyJSON []byte
		skyJSON, err = json.Marshal(sky)
		if err != nil {
			fmt.Println(err)
		}
		nc.Notify(skyJSON)
	}
	gps.AddFilter("SKY", skyfilter)
	done := gps.Watch()

	// static file handling
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	// endpoint paths
	http.HandleFunc("/sse", handleSSE(nc))
	http.HandleFunc("/metrics", handleMetrics(nc))
	http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(":8555", nil))
	<-done
}
