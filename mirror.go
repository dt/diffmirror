package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
)

type Mirror struct {
	settings *Settings

	queue    chan []byte
	reporter *DiffReporter
	stats    *Stats

	working *sync.WaitGroup
}

func NewMirror(s *Settings) *Mirror {
	m := new(Mirror)
	m.settings = s

	m.queue = make(chan []byte, 100)

	m.stats = NewStats(s.printStats, s.graphiteHost, s.graphitePrefix)
	m.reporter = NewDiffReporter(s, m.stats)

	for i := 0; i < s.workers; i++ {
		go m.worker()
	}

	if s.trackWork {
		m.working = new(sync.WaitGroup)
	}

	return m
}

func (m *Mirror) worker() {
	for {
		m.unpackAndHandle(<-m.queue)
	}
}

func (m *Mirror) unpackAndHandle(raw []byte) {
	if m.working != nil {
		m.working.Add(1)
		defer m.working.Done()
	}

	m.stats.Inc("mirror.requests")
	start := time.Now()

	reqA, _ := http.ReadRequest(bufio.NewReader(bytes.NewReader(raw)))
	reqB, _ := http.ReadRequest(bufio.NewReader(bytes.NewReader(raw)))
	m.mirror(reqA, reqB, raw)

	end := time.Now()

	m.stats.Timing("mirror.time", end.Sub(start))
}

func (m *Mirror) mirror(reqA, reqB *http.Request, raw []byte) {
	startA := time.Now()
	respA, statusA, errA := m.send(reqA, m.settings.hostA, m.settings.compareBodyOnly)
	rttA := time.Now().Sub(startA)

	startB := time.Now()
	respB, statusB, errB := m.send(reqB, m.settings.hostB, m.settings.compareBodyOnly)
	rttB := time.Now().Sub(startB)

	if errA != nil {
		log.Printf("error mirroring request: %s", errA)
	}

	if errB != nil {
		log.Printf("error mirroring request: %s", errB)
	}

	m.reporter.Compare(reqA, raw, statusA, statusB, respA, respB, errA, errB, rttA, rttB)
}

func (m *Mirror) send(r *http.Request, addr string, bodyOnly bool) (string, int, error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return "", -1, fmt.Errorf("error establishing tcp connection to %s: %s", addr, err)
	}
	defer c.Close()

	if err = r.Write(c); err != nil {
		return "", -1, fmt.Errorf("error initializing write to %s: %s", addr, err)
	}

	read := bufio.NewReader(c)
	resp, err := http.ReadResponse(read, nil)

	if err != nil {
		return "", -1, fmt.Errorf("error reading response from %s: %s", addr, err)
	}

	defer resp.Body.Close()

	if bodyOnly {
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		return string(contents), resp.StatusCode, nil
	} else {
		delete(resp.Header, "Date")
		respString, err := httputil.DumpResponse(resp, true)

		if err != nil {
			return "", -1, fmt.Errorf("error dumping response from %s: %s", addr, err)
		}

		return string(respString), resp.StatusCode, nil
	}
}
