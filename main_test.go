package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func server(head, body string) *httptest.Server {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-diffmirror-test", head)
		fmt.Fprintln(w, body)
	}))
	return s
}

func mockSettings(bodyOnly, ignoreErrors bool) *Settings {
	s := new(Settings)
	s.workers = 1
	s.trackWork = true
	s.compareBodyOnly = bodyOnly
	s.ignoreErrors = ignoreErrors
	return s
}

func runOne(t *testing.T, s *Settings, headA, headB, bodyA, bodyB string) *Stats {
	a := server(headA, bodyA)
	defer a.Close()

	b := server(headB, bodyB)
	defer b.Close()

	s.nameA = "a"
	s.hostA = strings.Replace(a.URL, "http://", "", -1)
	s.nameB = "b"
	s.hostB = strings.Replace(b.URL, "http://", "", -1)

	m := NewMirror(s)

	diffmirror := httptest.NewServer(MirrorServer{mirror: m})
	defer diffmirror.Close()

	log.SetOutput(ioutil.Discard)

	_, err := http.Get(diffmirror.URL)
	if err != nil {
		t.Fatal(err)
	}

	m.working.Wait()

	return m.stats
}

func expectStat(t *testing.T, s *Stats, stat string, expected int) {
	actual := int(s.GetCount(stat))
	if actual != expected {
		t.Errorf("Value '%s' expected to be %d but is %d", stat, expected, actual)
	}
}

func TestSame(t *testing.T) {
	s := runOne(t, mockSettings(false, false), "header", "header", "body", "body")
	expectStat(t, s, "diffing.total", 1)
	expectStat(t, s, "diffing.match", 1)
	expectStat(t, s, "diffing.diff", 0)
}

func TestIgnoredHeaderDiff(t *testing.T) {
	s := runOne(t, mockSettings(true, false), "headerA", "headerB", "body", "body")
	expectStat(t, s, "diffing.total", 1)
	expectStat(t, s, "diffing.match", 1)
	expectStat(t, s, "diffing.diff", 0)
}

func TestHeaderDiff(t *testing.T) {
	s := runOne(t, mockSettings(false, false), "headerA", "headerB", "body", "body")
	expectStat(t, s, "diffing.total", 1)
	expectStat(t, s, "diffing.match", 0)
	expectStat(t, s, "diffing.diff", 1)
}

func TestBodyDiff(t *testing.T) {
	s := runOne(t, mockSettings(true, false), "headerA", "headerB", "bodyA", "bodyB")
	expectStat(t, s, "diffing.total", 1)
	expectStat(t, s, "diffing.match", 0)
	expectStat(t, s, "diffing.diff", 1)
}

func TestBodyOOOFailDiff(t *testing.T) {
	m := mockSettings(true, false)
	s := runOne(t, m, "headerA", "headerB", "body", "ybod")
	expectStat(t, s, "diffing.total", 1)
	expectStat(t, s, "diffing.match", 0)
	expectStat(t, s, "diffing.diff", 1)
}

func TestBodyOOODiff(t *testing.T) {
	m := mockSettings(true, false)
	m.ignoreBodyOrder = true
	s := runOne(t, m, "headerA", "headerB", "body", "ybod")
	expectStat(t, s, "diffing.total", 1)
	expectStat(t, s, "diffing.match", 1)
	expectStat(t, s, "diffing.diff", 0)
}
