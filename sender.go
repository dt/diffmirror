package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

func sendAndTime(r *http.Request, addr string, bodyOnly bool) *MirrorResp {
	start := time.Now()
	res := send(r, addr, bodyOnly)
	res.rtt = time.Now().Sub(start)
	return &res
}

func send(r *http.Request, addr string, bodyOnly bool) MirrorResp {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return MirrorResp{err: fmt.Errorf("error establishing tcp connection to %s: %s", addr, err)}
	}
	defer c.Close()

	if err = r.Write(c); err != nil {
		return MirrorResp{err: fmt.Errorf("error initializing write to %s: %s", addr, err)}
	}

	read := bufio.NewReader(c)
	resp, err := http.ReadResponse(read, nil)

	if err != nil {
		return MirrorResp{err: fmt.Errorf("error reading response from %s: %s", addr, err)}
	}

	defer resp.Body.Close()

	if bodyOnly {
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		return MirrorResp{status: resp.StatusCode, payload: string(contents)}
	} else {
		delete(resp.Header, "Date")
		respString, err := httputil.DumpResponse(resp, true)

		if err != nil {
			return MirrorResp{err: fmt.Errorf("error dumping response from %s: %s", addr, err)}
		}

		return MirrorResp{status: resp.StatusCode, payload: string(respString)}
	}
}
