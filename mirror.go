package main

import (
	"bufio"
	"bytes"
	"log"
	"net/http"
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

type MirrorResp struct {
	status  int
	err     error
	payload string
	rtt     time.Duration
}

func (m *MirrorResp) isErr() bool {
	return m.err != nil || m.status/100 == 5
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
	backA := make(chan *MirrorResp)
	backB := make(chan *MirrorResp)

	go asyncSend(backA, reqA, m.settings.hostA, m.settings.compareBodyOnly)
	go asyncSend(backB, reqB, m.settings.hostB, m.settings.compareBodyOnly)

	resA := <-backA
	resB := <-backB

	if resA.err != nil {
		log.Printf("error mirroring request: %s", resA.err)
	}

	if resB.err != nil {
		log.Printf("error mirroring request: %s", resB.err)
	}

	m.reporter.Compare(reqA, raw, resA, resB)
}
