package main

import (
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/dt/gor_request_files/requestfiles"
)

type DiffReporter struct {
	total int64
	diff  int64

	settings *Settings

	stats     *Stats
	statNames StatNames

	// If writing out diffs, need a queue to serialize to a single writer.
	outQueue       chan []byte
	requestsWriter io.Writer
}

// Compute these once at startup to avoid allocating them every time
type StatNames struct {
	diff  string
	match string
	total string

	errA string
	errB string

	rttA string
	rttB string
}

func NewDiffReporter(s *Settings, stats *Stats) (d *DiffReporter) {
	r := new(DiffReporter)

	r.settings = s

	r.stats = stats

	r.statNames = StatNames{
		total: "diffing.total",
		match: "diffing.match",
		diff:  "diffing.diff",
		errA:  "diffing.err." + s.nameA,
		errB:  "diffing.err." + s.nameB,
		rttA:  "diffing.rtt." + s.nameA,
		rttB:  "diffing.rtt." + s.nameB,
	}

	if s.requestsFile != "" {
		r.outQueue = make(chan []byte, 100)
		r.requestsWriter = requestfiles.NewFileOutput(s.requestsFile)
		go r.writeDiffs()
	}

	return r
}

func (d *DiffReporter) writeDiffs() {
	for {
		req := <-d.outQueue
		d.requestsWriter.Write(req)
	}
}

func isErr(err error, status int) bool {
	return err != nil || status/100 == 5
}

func (d *DiffReporter) Compare(req *http.Request, raw []byte, statusA, statusB int, respA, respB string, rawErrA, rawErrB error, rttA, rttB time.Duration) {
	atomic.AddInt64(&d.total, 1)
	d.stats.Inc(d.statNames.total)

	errA := isErr(rawErrA, statusA)
	errB := isErr(rawErrB, statusB)

	if errA {
		d.stats.Inc(d.statNames.errA)
	} else {
		d.stats.Timing(d.statNames.rttA, rttA)
	}

	if errB {
		d.stats.Inc(d.statNames.errB)
	} else {
		d.stats.Timing(d.statNames.rttB, rttB)
	}

	if (errA && errB) || (d.settings.ignoreErrors && (errA || errB)) {
		return
	}

	if !errA && !errB && respA == respB {
		d.stats.Inc(d.statNames.match)
		return
	}

	atomic.AddInt64(&d.diff, 1)
	d.stats.Inc(d.statNames.diff)

	sizeA := len(respA)
	sizeB := len(respB)

	limit := sizeA
	if sizeA > sizeB {
		limit = sizeB
	}

	i := 0
	for i < limit {
		if respA[i] != respB[i] {
			break
		}
		i++
	}

	start := 0
	if i > 100 {
		start = i - 100
	}

	end := i + 100
	if end > limit {
		end = limit
	}

	log.Printf(
		`[DIFF %d/%d] %s %s [status: %d v %d size: %d v %d (%d) time: %dms vs %dms (%d)]
		bytes %d - %d
		######## %s ########
		%s
		######## %s ########
		%s
		####################
		`,
		atomic.LoadInt64(&d.diff),
		atomic.LoadInt64(&d.total),
		req.Method,
		req.RequestURI,
		statusA, statusB,
		sizeA, sizeB, sizeA-sizeB,
		ms(rttA), ms(rttB), ms(rttA-rttB),
		start,
		end,
		d.settings.nameA,
		string(respA[start:end]),
		d.settings.nameB,
		string(respB[start:end]),
	)

	if d.requestsWriter != nil {
		d.outQueue <- raw
	}

}

func ms(d time.Duration) time.Duration {
	return d / time.Millisecond
}
