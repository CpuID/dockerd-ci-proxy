package main

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"time"
)

// Man-in-the-middle the Docker API call, to insert labels
// where appropriate to the HTTP request
//
// HTTP Response is not a part of this function, handled by io.Copy independently, request only
// Executed once for each HTTP request
//
// input = HTTP request data from the client
// output = HTTP request data to the Docker daemon
// TODO: do io.Reader/io.Writer need to be pointers?
// Cannot return errors here, run in a goroutine (in parallel to response path / io.Copy)
// Would be nice if we didn't have to buffer the incoming HTTP requests here, but I don't think we can avoid it...
func mitmDockerApiCall(input io.Reader, output io.Writer) {
	buffer := bufio.NewReader(input)
	for i := 0; i <= 10; i++ {
		log.Printf("STARTING REQUEST, SIZE: %d", buffer.Size())
		time.Sleep(50 * time.Millisecond)
	}
	var data []byte
	bytes, err := buffer.Read(data)
	log.Printf("REQUEST, bytes: %d, data: %s", bytes, string(data))
	req, err := http.ReadRequest(buffer)
	if err != nil {
		log.Printf("Error parsing Docker API client request: %s", err.Error())
		return
	}
	log.Printf("%+v\n", req)
}
