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

// Takes JSON in, returns JSON
func injectLabelToPostBody(input string) string {
	segments := []string{}
	// So we can put the trailing whitespace back if it existed on input
	trailing_whitespace := ""
	switch input[len(input)-1:] {
	case "\r":
		trailing_whitespace = "\r"
	case "\n":
		trailing_whitespace = "\n"
	}
	fmt.Printf("use input: %s\n", strings.TrimSpace(input[1:len(input)-2]))
	for _, v := range strings.Split(strings.TrimSpace(input[1:len(input)-2]), ",") {
		if len(v) >= 9 && v[0:9] == `"Labels":` {
			// Found the Labels segment
			if v[len(v)-2:] == `{}` {
				// Labels is currently empty, remove }
				v = v[0 : len(v)-1]
				fmt.Printf("labels is empty\n")
			} else {
				// Labels is currently non-empty, remove }, add a comma
				v = fmt.Sprintf("%s,", v[0:len(v)-1])
				fmt.Printf("labels is non-empty\n")
			}
			// Append the custom label + } suffix.
			v = fmt.Sprintf("%s\"%s\":\"%s\"}", v, docker_label_name, docker_label_value)
			fmt.Printf("do the labels append\n")
		}
		fmt.Printf("segment: %s\n", v)
		segments = append(segments, v)
	}
	//fmt.Printf("%+v\n", segments)
	return fmt.Sprintf("{%s}%s", strings.Join(segments, ","), trailing_whitespace)
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

	// Handle parent-cgroup + label injection
	// TODO: change this to look at the URI suffix, possibly "/create$"
	if r.Method == "POST" && string(body[0:1]) == "{" {
		// POST with JSON body
		// Not going down the road of parsing out the JSON as native types, due to the sheer volume of types + API versions that would need to be handled.
		// Introspect the JSON as a string and inject in the correct location instead.
		fmt.Printf("-----------\nBEFORE BODY (len %d):\n'%v'\n\n==--==\n", len(body), body)
		body = []byte(injectLabelToPostBody(string(body)))
		fmt.Printf("-----------\nAFTER BODY (len %d):\n'%v'\n\n==--==\n", len(body), body)
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
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("MITM -- Error generating upstream request: %s\n", err.Error())))
		return
	}
	// Most POST requests should have Content-Type: application/json, except for "docker import" which looks to use Content-Type: text/plain
	ureq.Header = r.Header
	// From docs:
	// For incoming requests, the Host header is promoted to the
	// Request.Host field and removed from the Header map.
	ureq.Host = r.Host
	uresp, err := httpc.Do(ureq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("MITM -- Error on upstream request: %s\n", err.Error())))
		return
	}
	log.Printf("MITM -- Received upstream response: %+v\n", uresp)
	// TODOLATER: biggest place this will get nasty otherwise is on "docker export" operations, in terms of buffering a full image (memory footprint).
	// If we could only do an io.Copy here instead on the response...
	defer uresp.Body.Close()
	ubody, err := ioutil.ReadAll(uresp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("MITM -- Error reading upstream response body: %s\n", err.Error())))
		return
	}
	for hk, hv := range uresp.Header {
		for _, hv2 := range hv {
			w.Header().Set(hk, hv2)
		}
	}
	w.WriteHeader(uresp.StatusCode)
	//fmt.Fprintf(w, strings.TrimSpace(string(ubody)))
	fmt.Fprintf(w, string(ubody))
	log.Printf("MITM -- Response sent to client.\n")
}
