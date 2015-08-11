package main

import (
	"net/http"
	"strings"
)

type Bucketer interface {
	Bucket(r *http.Request, payload []byte) string
}

type BodySlicer struct {
	start int
	end   int
}

func (s *BodySlicer) Bucket(r *http.Request, payload []byte) string {
	end := s.end
	if end >= len(payload) {
		end = len(payload) - 1
	}
	return string(payload[s.start:end])
}

type PathSlicer struct {
	start int
	end   int
}

func (s *PathSlicer) Bucket(r *http.Request, payload []byte) string {
	parts := strings.Split(r.RequestURI, "/")
	start := s.start
	if start >= len(parts) {
		return ""
	}

	end := s.end
	if end >= len(parts) {
		end = len(parts) - 1
	}

	return strings.Join(parts[start:end], "_")
}
