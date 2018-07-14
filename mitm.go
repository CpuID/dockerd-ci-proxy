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

	upstream_req_content_type := ""
	if r.Method == "POST" {
		if r.Header.Get("Content-Type") != "" {
			upstream_req_content_type = r.Header.Get("Content-Type")
		} else {
			log.Printf("MITM -- Received POST request to URI '%s' without Content-Type header, cannot proxy.\n", r.URL.String())
			// TODO: error to w + return
			return
		}
		// TODO: parse out body, ensure we apply the correct labels/parent cgroup respectively. JSON?
	}

	// Credit: https://gist.github.com/teknoraver/5ffacb8757330715bcbcc90e6d46ac74#file-unixhttpc-go-L27
	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", h.TargetSocket)
			},
		},
	}

	// Upstream response
	var ur *http.Response
	// TODO: conditional on HTTP method types
	log.Printf("MITM -- Make upstream request...\n")
	switch r.Method {
	case "GET":
		ur, err = httpc.Get("http://unix" + r.URL.String())
	case "POST":
		// Most should have Content-Type: application/json, except for "docker import" which looks to use Content-Type: text/plain
		// Use whatever the received request Content-Type is here.
		// TODO: use modified body here
		ur, err = httpc.Post("http://unix"+r.URL.String(), upstream_req_content_type, strings.NewReader(string(body)))
	default:
		log.Printf("MITM -- Unsupported HTTP method '%s', cannot passthrough", r.Method)
		// TODO: error to w + return
		return
	}
	if err != nil {
		log.Printf("MITM -- Error on upstream request: %s\n", err.Error())
		// TODO: error to w + return
		return
	}
	log.Printf("MITM -- Received upstream response: %+v\n", ur)
	// TODO: proxy response through to ResponseWriter? can we io.Copy?
	// Biggest place this will get nasty otherwise is on "docker export" operations, in terms of buffering a full image (memory footprint).
	defer ur.Body.Close()
	ubody, err := ioutil.ReadAll(ur.Body)
	if err != nil {
		log.Printf("MITM -- Error reading upstream response body: %s\n", err.Error())
		// TODO: error to w + return
		return
	}
	fmt.Fprintf(w, string(ubody))
	//fmt.Fprintf(w, "hello")
	log.Printf("MITM -- Response sent to client.\n")
}
