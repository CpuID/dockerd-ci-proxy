package main

import (
	//"bufio"
	//"fmt"
	"log"
	"net"
	//"net/http"
)

// Credit: https://gist.github.com/hakobe/6f70d69b8c5243117787fd488ae7fbf2
func mockDockerDaemonConn(c net.Conn) {
	if debug_mode >= 2 {
		log.Printf("Mock Docker -- New Connection.\n")
	}
	// TODOLATER: use this instead?
	/*
			req, err := http.ReadRequest(bufio.NewReader(c))
			if err != nil {
				the_error := fmt.Sprintf("Failed to read request: %s", err.Error())
				http_error := fmt.Sprintf("HTTP/1.1 500 Internal Server Error\r\nContent-Length: %d\r\n\r\n%s", len(the_error), the_error)
				_, err := c.Write([]byte(http_error))
				if err != nil {
					log.Fatal("Cannot write: ", err)
				}
			}
		log.Printf("Mock Docker -- HTTP Request: %+v\n", req)
	*/
	//var data []byte
	//for {
	if debug_mode >= 2 {
		log.Printf("Mock Docker -- Receiving....\n")
	}
	// TODOLATER: use Content-Length to determine max size here...
	buf := make([]byte, 2048)
	nr, err := c.Read(buf)
	if err != nil {
		return
	}
	if debug_mode >= 2 {
		log.Printf("Mock Docker -- Received partial: %s\n", string(buf))
	}
	data := buf[0:nr]
	//}
	if debug_mode >= 2 {
		log.Printf("Mock Docker -- Server got full request: %s\n", string(data))
	}
	// TODOLATER: remove the use of a global here...
	last_received_request_to_mocked_daemon = string(data)
	response := "HTTP/1.1 200 OK\r\nApi-Version: 1.31\r\nContent-Type: application/json\r\nDocker-Experimental: false\r\nOstype: linux\r\nServer: Docker/17.07.0-ce (linux)\r\nDate: Sat, 14 Jul 2018 03:17:00 GMT\r\nContent-Length: 3\r\n\r\n[]\r"
	_, err = c.Write([]byte(response))
	if err != nil {
		log.Fatal("Cannot write: ", err)
	}
	last_sent_response_from_mocked_daemon = response
	if debug_mode >= 2 {
		log.Printf("Mock Docker -- Response sent.\n")
	}
	err = c.Close()
	if err != nil {
		log.Fatal("Cannot close connection: ", err)
	}
	if debug_mode >= 2 {
		log.Printf("Mock Docker -- Connection closed successfully.\n")
	}
}

func mockDockerDaemon(l net.Listener) {
	for {
		fd, err := l.Accept()
		if err != nil {
			log.Fatalf("Mock Docker -- Accept error: %s\n", err)
		}

		go mockDockerDaemonConn(fd)
	}
}
