package main

import (
	"log"
	"net"
	"os"
	"sync"
	"testing"
)

// TODOLATER: this is pretty ugly, but we will define the received mocked request as a global, and verify it in the coverage
var last_received_request_to_mocked_daemon string
var last_sent_response_from_mocked_daemon string

// One test at a time, since we are using some global hackery above
var mocked_docker_daemon_mutex sync.Mutex

// docker client
var dc net.Conn

func init() {
	// Enable "some" debug by default, change this to 2 if you need more and run "go test"
	debug_mode = 2
}

func TestMain(m *testing.M) {
	// Start up a mocked Docker daemon unix socket, to receive calls on.
	// TODOLATER: do we use httptest.NewServer() here instead? more elegant + we can use non-global variable state to verify received requests...?
	mocked_docker_daemon_socket_path := "/tmp/mock_docker.sock"
	if _, err := os.Stat(mocked_docker_daemon_socket_path); err == nil {
		os.Remove(mocked_docker_daemon_socket_path)
	}
	ml, err := net.Listen("unix", mocked_docker_daemon_socket_path)
	if err != nil {
		log.Fatalf(err.Error())
	}
	mocked_docker_daemon, err := newStoppableUnixListener(ml)
	defer os.Remove(mocked_docker_daemon_socket_path)
	if err != nil {
		log.Fatalf(err.Error())
	}
	go mockDockerDaemon(mocked_docker_daemon)
	defer mocked_docker_daemon.Stop()

	// Define the CI Proxy socket path (mocked for test coverage)
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
	dc, err = net.Dial("unix", mocked_proxy_socket_path)
	if err != nil {
		panic(err.Error())
	}

	// Set mock Docker label name/value
	docker_label_name = "Created-Via-Mock-By"
	docker_label_value = "dockerd-ci-proxy-test"

	// Do the tests
	exitcode := m.Run()

	// Stop the proxy
	stopDockerProxy(&wg, &docker_proxy)

	os.Exit(exitcode)
}
