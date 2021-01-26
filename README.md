YoGoGPS
=======

A simple web server that serves up GPSD SKY and TPV data via SSE (ServerSideEvents).

Environment variables:
- `GPSD_SERVER` - The IP and port of the gpsd process. (default: localhost:2947)

Commands:
The simple compile and run in one:
`$ go run yogogps.go`
Or compile and then run the binary:
```shell
$ go build yogogps.go
$ ./yogogps.go
```
Note: `go install` won't work unless you then run it from the current project directory due to the required template and static files.
