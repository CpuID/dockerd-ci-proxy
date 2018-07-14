package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
)

type mitmHttpHandler struct {
	TargetSocket string
}

func (h *mitmHttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("MITM -- New request received:\n")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("MITM -- Error reading HTTP request body: %s\n", err.Error())
	}
	log.Printf("%s %s\n", r.Method, r.URL.String())
	log.Printf("MITM -- Headers: %+v\n", r.Header)
	log.Printf("MITM -- Body: %s\n", body)
	log.Printf("----------\n")

	if r.Method == "POST" {
		// TODO: parse out body, ensure we apply the correct labels/parent cgroup respectively. JSON?
	}

	log.Printf("MITM -- Make upstream request...\n")

	// Ensure compression matches original request
	disable_compression := false
	if r.Header.Get("Accept-Encoding") == "" {
		disable_compression = true
	}

	// Credit: https://gist.github.com/teknoraver/5ffacb8757330715bcbcc90e6d46ac74#file-unixhttpc-go-L27
	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", h.TargetSocket)
			},
			DisableCompression: disable_compression,
		},
	}

	// TODO: use pre-parsed out body here for POST requests
	ureq, err := http.NewRequest(r.Method, "http://unix"+r.URL.String(), strings.NewReader(string(body)))
	if err != nil {
		log.Printf("MITM -- Error generating upstream request: %s\n", err.Error())
		// TODO: error to w + return
	}
	// Most POST requests should have Content-Type: application/json, except for "docker import" which looks to use Content-Type: text/plain
	ureq.Header = r.Header
	// From docs:
	// For incoming requests, the Host header is promoted to the
	// Request.Host field and removed from the Header map.
	ureq.Host = r.Host
	uresp, err := httpc.Do(ureq)
	if err != nil {
		log.Printf("MITM -- Error on upstream request: %s\n", err.Error())
		// TODO: error to w + return
		return
	}
	log.Printf("MITM -- Received upstream response: %+v\n", uresp)
	// TODO: proxy response through to ResponseWriter? can we io.Copy?
	// Biggest place this will get nasty otherwise is on "docker export" operations, in terms of buffering a full image (memory footprint).
	defer uresp.Body.Close()
	ubody, err := ioutil.ReadAll(uresp.Body)
	if err != nil {
		log.Printf("MITM -- Error reading upstream response body: %s\n", err.Error())
		// TODO: error to w + return
		return
	}
	fmt.Fprintf(w, string(ubody))
	log.Printf("MITM -- Response sent to client.\n")
}
