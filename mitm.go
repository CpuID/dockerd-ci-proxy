package main

import (
	//"bufio"
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
	// TODO: implement
	log.Printf("NEW REQUEST:\n")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error on HTTP request: %s\n", err.Error())
	}
	log.Printf("%s %s\n", r.Method, r.URL.String())
	log.Printf("Headers: %+v\n", r.Header)
	log.Printf("Body: %s\n", body)
	log.Printf("----------\n")
	fmt.Fprintf(w, "HTTP 200 OK\n")

	log.Printf("mitm request, target socket: %s\n", h.TargetSocket)
	uc, err := net.Dial("unix", h.TargetSocket)
	defer uc.Close()
	if err != nil {
		log.Printf("%s - Failed to connect to UNIX Socket %s. Error: %s\n", app_general_name, h.TargetSocket, err.Error())
		// TODO: failure via fmt.Fprintf(w, ...)?
		return
	}

	_, err = uc.Write([]byte(fmt.Sprintf("%s\n", r.URL.String())))
	if err != nil {
		log.Printf("Error on upstream HTTP request: %s\n", err.Error())
	}
}

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
/*func mitmDockerApiCall(input io.Reader, output io.Writer) {
request, err := ioutil.ReadAll(input)
if err != nil {
	log.Printf("Error reading all of input: %s", err.Error())
	return
}
log.Printf("REQUEST: %s", request)
//buffer := bufio.NewReader(input)
*/
/*for i := 0; i <= 10; i++ {
	log.Printf("STARTING REQUEST, SIZE: %d", buffer.Size())
	time.Sleep(50 * time.Millisecond)
}
// Max 100 lines of request headers? we probably need to deal with the request body also...
for i := 0; i <= 50; i++ {
	line, err := buffer.ReadString('\n')
	log.Printf("LINE ERR: %s", err.Error())
	/*if err != nil {
		log.Printf("Error parsing Docker API client request (1): %s", err.Error())
		return
	}
	log.Printf("LINE: %s", line)
}
req, err := http.ReadRequest(buffer)
if err != nil {
	log.Printf("Error parsing Docker API client request: %s", err.Error())
	return
}
log.Printf("%+v\n", req)
*/
// }
