package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

type Settings struct {
	listen string

	workers int

	hostA string
	nameA string
	hostB string
	nameB string

	requestsFile string

	ignoreErrors    bool
	compareBodyOnly bool

	bucketer Bucketer

	printStats     bool
	graphiteHost   string
	graphitePrefix string

	trackWork bool
}

func extractAlias(s, defaultValue string) (string, string) {
	if strings.ContainsRune(s, '=') {
		p := strings.SplitN(s, "=", 2)
		return p[0], p[1]
	}
	return defaultValue, s
}

func getSettings() *Settings {
	s := new(Settings)

	flag.IntVar(&s.workers, "workers", 10, "number of worker threads")

	flag.StringVar(&s.requestsFile, "requestsfile", "", "filename in which to store requests that generated diffs")

	flag.BoolVar(&s.printStats, "stats", true, "print stats to console periodically")
	flag.StringVar(&s.graphiteHost, "graphite", "", "address of graphite receiver for stats")
	flag.StringVar(&s.graphitePrefix, "graphite-prefix", "", "prefix for graphite writes")

	flag.BoolVar(&s.ignoreErrors, "ignore-errors", true, "ignore network errors and 5xx responses")
	flag.BoolVar(&s.compareBodyOnly, "body-only", true, "compare only the body of responses (exclude headers)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\nUsage: %s [options] [aliasA=]hostA [aliasB=]hostB\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(flag.Args()) < 3 {
		flag.Usage()
		os.Exit(-1)
	}

	s.listen = flag.Arg(0)

	s.nameA, s.hostA = extractAlias(flag.Arg(1), "a")
	s.nameB, s.hostB = extractAlias(flag.Arg(2), "b")

	if !strings.ContainsRune(s.listen, ':') {
		s.listen = ":" + s.listen
	}

	return s
}

func main() {
	s := getSettings()
	m := NewMirror(s)

	srv := MirrorServer{mirror: m}

	log.Printf("Listening on %s and forwarding to %s (%s) and %s (%s).",
		s.listen,
		s.hostA, s.nameA,
		s.hostB, s.nameB,
	)

	log.Fatal(http.ListenAndServe(s.listen, srv))
}
