package main

import (
	//"bufio"
	"context"
	"fmt"
	//"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	//"time"
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

	// Credit: https://gist.github.com/teknoraver/5ffacb8757330715bcbcc90e6d46ac74#file-unixhttpc-go-L27
	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", h.TargetSocket)
			},
		},
	}

	// Upstream response
	//var ur *http.Response
	// TODO: conditional on HTTP methods
	log.Printf("MITM -- Make upstream request...\n")
	ur, err := httpc.Get("http://unix" + r.URL.String())
	if err != nil {
		log.Printf("MITM -- Error on upstream request: %s\n", err.Error())
	}
	log.Printf("MITM -- Received upstream response: %+v\n", ur)
	// TODO: proxy response through to ResponseWriter? can we io.Copy?
	// Biggest place this will get nasty otherwise is on "docker export" operations, in terms of buffering a full image (memory footprint).
	defer ur.Body.Close()
	ubody, err := ioutil.ReadAll(ur.Body)
	if err != nil {
		log.Printf("MITM -- Error reading upstream response body: %s\n", err.Error())
	}
	fmt.Fprintf(w, string(ubody))
	//fmt.Fprintf(w, "hello")
	log.Printf("MITM -- Response sent to client.\n")
}
