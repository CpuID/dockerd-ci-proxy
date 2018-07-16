package main

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// Because of the way the headers are reinjected into responses, the ordering will likely differ
// but they should be an identical list of key/value pairs otherwise.
// Take in the plaintext response, and sort the headers and return it so we can do a direct comparison.
// Simple enough solution right now to the issue... otherwise proxy test coverage is more of a PITA.
func sortResponseHeaders(resp []byte) []byte {
	status_line := []byte{}
	// string type is easier here for strings.Join helper functions etc.
	headers := []string{}
	body := []byte{}
	// First we need to split out the status line, the headers and the body.
	status_line_done := false
	headers_done := false
	for _, v := range strings.Split(string(resp), "\r\n") {
		if status_line_done == false {
			status_line = []byte(v)
			status_line_done = true
			continue
		}
		if headers_done == true {
			body = append(body, []byte(v)...)
			body = append(body, []byte("\r\n")...)
			continue
		}
		// Is this the end of the headers?
		if v == "" {
			headers_done = true
			continue
		}
		// Must be a header at this point
		headers = append(headers, v)
	}

	// Then sort the headers by key name
	sort.Strings(headers)

	// Strip final \r\n from body
	body = body[:len(body)-2]

	// Then rebuild the full response and return it
	return []byte(fmt.Sprintf("%s\r\n%s\r\n\r\n%s", status_line, strings.Join(headers, "\r\n"), body))
}

// Testing the test functionality, kind of needed to :)
func TestSortResponseHeaders(t *testing.T) {
	input := []byte("HTTP/1.1 200 OK\r\nHost: asdf\r\nContent-Type: application/json\r\nApi-Version: 1.31\r\n\r\nsome payload\nblah\n")
	expected_result := []byte("HTTP/1.1 200 OK\r\nApi-Version: 1.31\r\nContent-Type: application/json\r\nHost: asdf\r\n\r\nsome payload\nblah\n")
	result := sortResponseHeaders(input)
	if string(result) != string(expected_result) {
		t.Errorf("Expected (len %d):\n\n'%s'\n\nGot (len %d):\n\n'%s'", len(expected_result), string(expected_result), len(result), string(result))
	}
}