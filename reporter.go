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

	bucket *Bucketer

	stats             *Stats
	statNames         StatNames
	detailedStatNames map[string]*StatNames

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

func (d *DiffReporter) statNamesFor(bucket string) *StatNames {
	found, ok := d.detailedStatNames[bucket]
	if ok {
		return found
	}

	found = &StatNames{
		total: "diffing." + bucket + ".total",
		match: "diffing." + bucket + ".match",
		diff:  "diffing." + bucket + ".diff",
		errA:  "diffing." + bucket + ".err." + d.settings.nameA,
		errB:  "diffing." + bucket + ".err." + d.settings.nameB,
		rttA:  "diffing." + bucket + ".rtt." + d.settings.nameA,
		rttB:  "diffing." + bucket + ".rtt." + d.settings.nameB,
	}
	d.detailedStatNames[bucket] = found
	return found
}

func (d *DiffReporter) writeDiffs() {
	for {
		req := <-d.outQueue
		d.requestsWriter.Write(req)
	}
}

func (d *DiffReporter) Compare(req *http.Request, raw []byte, resA, resB *MirrorResp) {
	atomic.AddInt64(&d.total, 1)

	var bucketStats *StatNames
	if d.settings.bucketer != nil {
		bucket := d.settings.bucketer.Bucket(req, raw)
		if bucket != "" {
			bucketStats = d.statNamesFor(bucket)
		}
	}

	d.stats.Inc(d.statNames.total)
	if bucketStats != nil {
		d.stats.Inc(bucketStats.total)
	}

	errA := resA.isErr()
	errB := resB.isErr()

	if errA {
		d.stats.Inc(d.statNames.errA)
		if bucketStats != nil {
			d.stats.Inc(bucketStats.errA)
		}
	} else {
		d.stats.Timing(d.statNames.rttA, resA.rtt)
		if bucketStats != nil {
			d.stats.Timing(bucketStats.rttA, resB.rtt)
		}
	}

	if errB {
		d.stats.Inc(d.statNames.errB)
		if bucketStats != nil {
			d.stats.Inc(bucketStats.errB)
		}
	} else {
		d.stats.Timing(d.statNames.rttB, resB.rtt)
		if bucketStats != nil {
			d.stats.Timing(bucketStats.rttB, resB.rtt)
		}
	}

	if (errA && errB) || (d.settings.ignoreErrors && (errA || errB)) {
		return
	}

	if !errA && !errB && resA.payload == resB.payload {
		d.stats.Inc(d.statNames.match)
		if bucketStats != nil {
			d.stats.Inc(bucketStats.match)
		}
		return
	}

	atomic.AddInt64(&d.diff, 1)
	d.stats.Inc(d.statNames.diff)
	if bucketStats != nil {
		d.stats.Inc(bucketStats.diff)
	}
	sizeA := len(resA.payload)
	sizeB := len(resB.payload)

	limit := sizeA
	if sizeA > sizeB {
		limit = sizeB
	}

	i := 0
	for i < limit {
		if resA.payload[i] != resB.payload[i] {
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
		resA.status, resB.status,
		sizeA, sizeB, sizeA-sizeB,
		ms(resA.rtt), ms(resB.rtt), ms(resA.rtt-resB.rtt),
		start,
		end,
		d.settings.nameA,
		string(resA.payload[start:end]),
		d.settings.nameB,
		string(resB.payload[start:end]),
	)

	if d.requestsWriter != nil {
		d.outQueue <- raw
	}

}

func ms(d time.Duration) time.Duration {
	return d / time.Millisecond
}
