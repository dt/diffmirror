package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
)

type MirrorServer struct {
	mirror *Mirror
}

func (m MirrorServer) ServeHTTP(out http.ResponseWriter, req *http.Request) {
	req.URL.Scheme = "http"
	req.URL.Host = req.Host

	raw, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.Fatal(err)
	}
	select {
	case m.mirror.queue <- raw:
		m.mirror.stats.Gauge("mirror.queue", len(m.mirror.queue))
	default:
		m.mirror.stats.Inc("mirror.dropped")
	}

	fmt.Fprintf(out, "OK")
}
