package main

import (
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

// Credit: https://gist.github.com/hakobe/6f70d69b8c5243117787fd488ae7fbf2
func echoServerConn(c net.Conn) {
	log.Printf("echoServer: START\n")
	for {
		buf := make([]byte, 8)
		nr, err := c.Read(buf)
		if err != nil {
			return
		}

		data := buf[0:nr]
		println("Server got:", string(data))
		_, err = c.Write(data)
		if err != nil {
			log.Fatal("Write: ", err)
		}
	}
}

func echoServer(l net.Listener) {
	for {
		fd, err := l.Accept()
		if err != nil {
			log.Fatal("Accept error: ", err)
		}

		go echoServerConn(fd)
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
	go echoServer(mocked_docker_daemon)
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
	println("proceeding...")
	iter := 0
	for {
		println("sending...")
		_, err := c.Write([]byte("GET /v1.37/containers/json HTTP/1.1\r\nHost: docker\r\nUser-Agent: Docker-Client/18.03.1-ce (darwin)\r\n\r\n"))
		if err != nil {
			println(err.Error())
		}
		time.Sleep(1 * time.Second)
		iter = iter + 1
		if iter >= 5 {
			break
		}
	}

	stopDockerProxy(&wg, &docker_proxy)
}
