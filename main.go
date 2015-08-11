package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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

	bucketer   Bucketer
	bucketPath string
	bucketBody string

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

func intPair(s string) (int, int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("Must provide 'start:end'")
	}
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	end, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}

func getSettings() *Settings {
	s := new(Settings)

	flag.IntVar(&s.workers, "workers", 10, "number of worker threads")

	flag.StringVar(&s.requestsFile, "requestsfile", "", "filename in which to store requests that generated diffs")

	flag.BoolVar(&s.printStats, "stats", true, "print stats to console periodically")
	flag.StringVar(&s.graphiteHost, "graphite", "", "address of graphite receiver for stats")
	flag.StringVar(&s.bucketPath, "bucket-by-path-parts", "", "start:end offsets for path parts (split by /) for bucketing")
	flag.StringVar(&s.bucketBody, "bucket-by-body-slice", "", "start:end offsets to slice from the body for bucketing")
	flag.StringVar(&s.graphitePrefix, "graphite-prefix", "", "prefix for graphite writes")

	flag.BoolVar(&s.ignoreErrors, "ignore-errors", true, "ignore network errors and 5xx responses")
	flag.BoolVar(&s.compareBodyOnly, "body-only", true, "compare only the body of responses (exclude headers)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\nUsage: %s [options] [aliasA=]hostA [aliasB=]hostB\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if s.bucketBody != "" && s.bucketPath != "" {
		log.Fatalln("only one bucketing may be used")
	}

	if s.bucketBody != "" {
		start, end, err := intPair(s.bucketBody)
		if err != nil {
			log.Fatalln(err)
		}
		s.bucketer = &BodySlicer{start, end}
	}

	if s.bucketPath != "" {
		start, end, err := intPair(s.bucketBody)
		if err != nil {
			log.Fatalln(err)
		}
		s.bucketer = &PathSlicer{start, end}
	}

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
