package main

import (
	"bytes"
	"encoding/binary"
	"net/http"
	"strings"
)

type Bucketer interface {
	Bucket(r *http.Request, payload []byte) string
}

type RangeSlicer struct {
	start int
	end   int
}

func (s *RangeSlicer) Bucket(r *http.Request, payload []byte) string {
	end := s.end
	if end >= len(payload) {
		end = len(payload) - 1
	}
	return string(payload[s.start:end])
}

type CStringSlicer struct {
	start int
}

func (s *CStringSlicer) Bucket(r *http.Request, payload []byte) string {
	if s.start >= len(payload) {
		return ""
	}
	null := bytes.IndexByte(payload[s.start:], byte(0))
	if null == -1 {
		return ""
	}

	end := s.start + null - 1

	if end >= len(payload) {
		return ""
	}

	return string(payload[s.start:end])
}

type StrLenSlicer struct {
	pos int
}

func (s *StrLenSlicer) Bucket(r *http.Request, payload []byte) string {
	if s.pos+4 >= len(payload) {
		return ""
	}

	var rawSize uint32
	binary.Read(bytes.NewReader(payload[s.pos:s.pos+4]), binary.BigEndian, &rawSize)

	size := int(rawSize)

	start := s.pos + 4

	end := start + size
	if end >= len(payload) {
		end = len(payload) - 1
	}

	return string(payload[start:end])
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
