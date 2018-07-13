package main

import (
	"log"
	"net"
	"sync"
	"testing"
	"time"
)

// Credit: https://gist.github.com/hakobe/6f70d69b8c5243117787fd488ae7fbf2
func echoServer(c net.Conn) {
	for {
		buf := make([]byte, 512)
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

func DockerProxyMockTest(t *testing.T) {
	// Start up a mocked Docker daemon unix socket, to receive calls on.
	mocked_docker_daemon_socket_path := "/tmp/mock_docker.sock"
	ml, err := net.Listen("unix", mocked_docker_daemon_socket_path)
	if err != nil {
		t.Fatal(err.Error())
	}
	// mocked_docker_daemon_socket, err
	_, err = newStoppableUnixListener(ml)
	if err != nil {
		t.Fatal(err.Error())
	}
	// TODO: make mocked_docker_daemon_socket do something useful

	mocked_proxy_socket_path := "/tmp/mock_docker_proxy.sock"

	// Start up the proxy
	docker_proxy := dockerProxy{ListenSocket: mocked_proxy_socket_path, TargetSocket: mocked_docker_daemon_socket_path}
	var wg sync.WaitGroup
	ready := make(chan int)
	wg.Add(1)
	go docker_proxy.runProxy(&wg, ready)
	<-ready

	// Make a connection to the proxy, to fire off some commands
	c, err := net.Dial("unix", mocked_proxy_socket_path)
	if err != nil {
		panic(err.Error())
	}
	for {
		_, err := c.Write([]byte("hi\n"))
		if err != nil {
			println(err.Error())
		}
		time.Sleep(1e9)
	}

}
