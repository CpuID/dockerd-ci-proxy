package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
)

type mitmHttpHandler struct {
	TargetSocket string
}

// Takes JSON in, returns JSON
func injectLabelAndCgroupParentToPostBody(input string, cgroup_parent string) string {
	segments := []string{}
	// So we can put the trailing whitespace back if it existed on input
	re := regexp.MustCompile("(\\r|\\n)+$")
	trailing_whitespace := re.FindString(input)
	// Remove the trailing whitespace we found then proceed
	input = input[:len(input)-len(trailing_whitespace)]
	for _, v := range strings.Split(input[1:len(input)-1], ",") {
		if len(v) >= 9 && v[0:9] == `"Labels":` {
			// Found the Labels segment
			if v[len(v)-2:] == `{}` {
				// Labels is currently empty, remove }
				v = v[0 : len(v)-1]
			} else {
				// Labels is currently non-empty, remove }, add a comma
				v = fmt.Sprintf("%s,", v[0:len(v)-1])
			}
			// Append the custom label + } suffix.
			v = fmt.Sprintf("%s\"%s\":\"%s\"}", v, docker_label_name, docker_label_value)
		} else if len(v) >= 15 && v[0:15] == `"CgroupParent":` && len(cgroup_parent) > 0 {
			// Found the CgroupParent segment
			// TODOLATER: do we want to error out if one is already set?
			v = fmt.Sprintf("\"CgroupParent\":\"%s\"", cgroup_parent)
		}
		segments = append(segments, v)
	}
	return fmt.Sprintf("{%s}%s", strings.Join(segments, ","), trailing_whitespace)
}

func (h *mitmHttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if debug_mode >= 1 {
		log.Printf("%s -=- MITM -- New request received:\n", app_code_name)
		log.Printf("%s -=- MITM -- %s %s\n", app_code_name, r.Method, r.URL.String())
		log.Printf("%s -=- MITM -- Headers: %+v\n", app_code_name, r.Header)
		log.Printf("%s -=- ----------\n", app_code_name)
	}

	// Handle parent-cgroup + label injection
	// Selected URI suffixes only for now.
	var body []byte
	var err error
	uri_re := regexp.MustCompile(`\/build|\/(containers|volumes|networks|plugins|services|secrets|configs)\/create$`)
	if r.Method == "POST" && uri_re.MatchString(r.URL.Path) == true {
		if debug_mode >= 1 {
			log.Printf("%s -=- MITM -- Method/URI match, parsing out request body.\n", app_code_name)
		}
		body, err = ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("%s -=- MITM -- Cannot read API request body: %s", app_code_name, err.Error())))
			return
		}
		if debug_mode >= 1 {
			log.Printf("%s -=- MITM -- Body: %s\n", app_code_name, body)
			log.Printf("%s -=- ----------\n", app_code_name)
		}
		if string(body[0:1]) != "{" {
			// Non-JSON request body, this should never happen based on the API docs (for the method list above)?
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("%s -=- MITM -- Non-JSON body detected for API call? This should not occur", app_code_name)))
			return
		}
		// POST with JSON body
		// Not going down the road of parsing out the JSON as native types, due to the sheer volume of types + API versions that would need to be handled.
		// Introspect the JSON as a string and inject in the correct location instead.
		body = []byte(injectLabelAndCgroupParentToPostBody(string(body), docker_cgroup_parent))
	}

	if debug_mode >= 2 {
		log.Printf("%s -=- MITM -- Make upstream request...\n", app_code_name)
	}

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

	var ureq_body io.Reader
	if len(body) > 0 {
		// Body was parsed out above, feed it in modified.
		ureq_body = strings.NewReader(string(body))
	} else {
		// Feed in the reader from upstream request instead (passthrough)
		ureq_body = r.Body
	}
	ureq, err := http.NewRequest(r.Method, "http://unix"+r.URL.String(), ureq_body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("%s -=- MITM -- Error generating upstream request: %s\n", app_code_name, err.Error())))
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
		w.Write([]byte(fmt.Sprintf("%s -=- MITM -- Error on upstream request: %s\n", app_code_name, err.Error())))
		return
	}
	if debug_mode >= 2 {
		log.Printf("%s -=- MITM -- Received upstream response: %+v\n", app_code_name, uresp)
	}
	defer uresp.Body.Close()
	for hk, hv := range uresp.Header {
		for _, hv2 := range hv {
			w.Header().Set(hk, hv2)
			// TODOLATER: modify test coverage to not break when adding this.
			//w.Header().Set("X-Served-By", "dockerd-ci-proxy")
		}
	}
	w.WriteHeader(uresp.StatusCode)
	// Passthrough responses, so that "docker export" operations should be nice and smooth (large payloads)
	// No modifications are made here anyway.
	io.Copy(w, uresp.Body)
	if debug_mode >= 1 {
		log.Printf("%s -=- MITM -- Response sent to client.\n", app_code_name)
		log.Printf("%s -=- ==========\n", app_code_name)
	}
}
