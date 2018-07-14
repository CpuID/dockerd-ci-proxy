package main

import (
	"log"
	"net"
	"os"
	"sync"
	"testing"
)

// Credit: https://gist.github.com/hakobe/6f70d69b8c5243117787fd488ae7fbf2
func mockDockerDaemonConn(c net.Conn) {
	log.Printf("Mock Docker -- New Request Received.\n")
	for {
		buf := make([]byte, 512)
		nr, err := c.Read(buf)
		if err != nil {
			return
		}

		data := buf[0:nr]
		println("Server got:", string(data))
		response := "HTTP/1.1 200 OK\r\nApi-Version: 1.31\r\nContent-Type: application/json\r\nDocker-Experimental: false\r\nOstype: linux\r\nServer: Docker/17.07.0-ce (linux)\r\nDate: Sat, 14 Jul 2018 03:17:00 GMT\r\nContent-Length: 3\r\n\r\n[]\r\n"
		_, err = c.Write([]byte(response))
		if err != nil {
			log.Fatal("Cannot write: ", err)
		}
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

func TestDockerProxyMock(t *testing.T) {
	// Start up a mocked Docker daemon unix socket, to receive calls on.
	mocked_docker_daemon_socket_path := "/tmp/mock_docker.sock"
	if _, err := os.Stat(mocked_docker_daemon_socket_path); err == nil {
		os.Remove(mocked_docker_daemon_socket_path)
	}
	ml, err := net.Listen("unix", mocked_docker_daemon_socket_path)
	if err != nil {
		t.Fatal(err.Error())
	}
	mocked_docker_daemon, err := newStoppableUnixListener(ml)
	defer os.Remove(mocked_docker_daemon_socket_path)
	if err != nil {
		t.Fatal(err.Error())
	}
	go mockDockerDaemon(mocked_docker_daemon)
	defer mocked_docker_daemon.Stop()

	mocked_proxy_socket_path := "/tmp/mock_docker_proxy.sock"
	if _, err := os.Stat(mocked_proxy_socket_path); err == nil {
		os.Remove(mocked_proxy_socket_path)
	}

	// Start up the proxy
	docker_proxy := dockerProxy{}
	var wg sync.WaitGroup
	ready := make(chan int)
	startDockerProxy(&wg, &docker_proxy, ready, mocked_docker_daemon_socket_path, mocked_proxy_socket_path)
	defer os.Remove(mocked_proxy_socket_path)

	// Make a connection to the proxy, to fire off some commands
	c, err := net.Dial("unix", mocked_proxy_socket_path)
	if err != nil {
		panic(err.Error())
	}
	// Fire off the same command twice, "good enough"
	for i := 0; i < 2; i++ {
		log.Printf("====================================================================\n")
		log.Printf("====================================================================\n")
		log.Printf("Client -- Sending request...\n")
		_, err := c.Write([]byte("GET /v1.37/containers/json HTTP/1.1\r\nHost: docker\r\nUser-Agent: Docker-Client/18.03.1-ce (darwin)\r\n\r\n"))
		if err != nil {
			println(err.Error())
		}
		//time.Sleep(500 * time.Millisecond)
		buf := make([]byte, 128)
		_, err = c.Read(buf)
		if err != nil {
			return
		}
		log.Printf("Client -- Response received: %s\n", buf)
	}
	log.Printf("====================================================================\n")
	log.Printf("====================================================================\n")

	stopDockerProxy(&wg, &docker_proxy)
}
