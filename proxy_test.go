package main

import (
	"testing"
)

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
	mocked_docker_daemon_socket := ""
	mocked_proxy_socket := ""

	// Start up a mocked Docker daemon unix socket, to receive calls on.
	// TODO: implement, listening on mocked_docker_daemon_socket

	// Start up the proxy
	docker_proxy := dockerProxy{ListenSocket: mocked_proxy_socket, TargetSocket: mocked_docker_daemon_socket}
	var wg sync.WaitGroup
	ready := make(chan int)
	wg.Add(1)
	go docker_proxy.runProxy(&wg, ready)
	<-ready

	// Make a connection to the proxy, to fire off some commands
	c, err := net.Dial("unix", "", "/tmp/echo.sock")
	if err != nil {
		panic(err.String())
	}
	for {
		_, err := c.Write([]byte("hi\n"))
		if err != nil {
			println(err.String())
		}
		time.Sleep(1e9)
	}

}
