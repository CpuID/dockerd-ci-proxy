package main

import (
	"log"
	"net"
)

// Credit: https://gist.github.com/hakobe/6f70d69b8c5243117787fd488ae7fbf2
func mockDockerDaemonConn(c net.Conn) {
	log.Printf("Mock Docker -- New Request Received.\n")
	for {
		// NOTE: the real upstream won't be doing 2048b blocks...
		// should we be using httptest.NewServer() instead?
		buf := make([]byte, 2048)
		nr, err := c.Read(buf)
		if err != nil {
			return
		}

		data := buf[0:nr]
		println("Server got:", string(data))
		// TODOLATER: remove the use of a global here...
		last_received_request_to_mocked_daemon = string(data)
		response := "HTTP/1.1 200 OK\r\nApi-Version: 1.31\r\nContent-Type: application/json\r\nDocker-Experimental: false\r\nOstype: linux\r\nServer: Docker/17.07.0-ce (linux)\r\nDate: Sat, 14 Jul 2018 03:17:00 GMT\r\nContent-Length: 3\r\n\r\n[]\r"
		_, err = c.Write([]byte(response))
		if err != nil {
			log.Fatal("Cannot write: ", err)
		}
		last_sent_response_from_mocked_daemon = response
	}
	log.Printf("Mock Docker -- Response sent.\n")
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
