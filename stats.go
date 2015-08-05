package main

import (
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/cyberdelia/go-metrics-graphite"
	"github.com/rcrowley/go-metrics"
)

type Stats struct {
	registry metrics.Registry
	lock     sync.Mutex
}

func (t *Stats) Gauge(name string, value int) {
	metrics.GetOrRegisterGauge(name, t.registry).Update(int64(value))
}

func (t *Stats) Inc(stat string) {
	metrics.GetOrRegisterCounter(stat, t.registry).Inc(1)
}

func (t *Stats) GetCount(stat string) int64 {
	c := t.registry.Get(stat)
	if c == nil {
		return 0
	}
	return c.(metrics.Counter).Count()
}

func (t *Stats) Timing(stat string, d time.Duration) {
	metrics.GetOrRegisterTimer(stat, t.registry).Update(d)
}

func NewStats(sendToConsole bool, sendToGraphite, graphitePrefix string) *Stats {
	s := new(Stats)
	s.registry = metrics.NewRegistry()

	if sendToGraphite != "" {
		log.Println("Stats reporting to graphite: ", sendToGraphite)
		addr, _ := net.ResolveTCPAddr("tcp", sendToGraphite)
		go graphite.Graphite(s.registry, time.Second*5, graphitePrefix, addr)
	}

	if sendToConsole {
		log.Println("Stats reporting enabled...")
		go metrics.Log(s.registry, time.Minute, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))
	}
	return s
}
