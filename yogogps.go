package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/stratoberry/go-gpsd"
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
		var tpvJSON []byte
		tpvJSON, err = json.Marshal(tpv)
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
	http.HandleFunc("/", home)
	http.HandleFunc("/sse", handleSSE(nc))
	log.Fatal(http.ListenAndServe(":8555", nil))
	<-done
}
