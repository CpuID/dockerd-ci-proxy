package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

// TODOLATER: this is pretty ugly, but we will define the received mocked request as a global, and verify it in the coverage
var last_received_request_to_mocked_daemon string
var last_sent_response_from_mocked_daemon string

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
		last_received_request_to_mocked_daemon = string(data)
		response := "HTTP/1.1 200 OK\r\nApi-Version: 1.31\r\nContent-Type: application/json\r\nDocker-Experimental: false\r\nOstype: linux\r\nServer: Docker/17.07.0-ce (linux)\r\nDate: Sat, 14 Jul 2018 03:17:00 GMT\r\nContent-Length: 3\r\n\r\n[]\r\n"
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

func TestDockerProxyMock(t *testing.T) {
	// Start up a mocked Docker daemon unix socket, to receive calls on.
	// TODO: do we use httptest.NewServer() here instead? more elegant + we can use non-global variable state to verify received requests...?
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
	c, err := net.Dial("unix", mocked_proxy_socket_path)
	if err != nil {
		panic(err.Error())
	}
	// Fire off 2 x "docker ps" executions, validate standard passthrough behaviour
	for i := 0; i < 2; i++ {
		log.Printf("====================================================================\n")
		log.Printf("====================================================================\n")
		ps_req_payload := "GET /v1.37/containers/json HTTP/1.1\r\nHost: docker\r\nUser-Agent: Docker-Client/18.03.1-ce (darwin)\r\n\r\n"
		log.Printf("Client -- Sending request: %s", ps_req_payload)
		_, err := c.Write([]byte(ps_req_payload))
		if err != nil {
			println(err.Error())
		}
		time.Sleep(100 * time.Millisecond)
		if ps_req_payload != last_received_request_to_mocked_daemon {
			t.Errorf("Expected request (len %d):\n\n%s\n\nGot request (len %d):\n\n%s\n", len(ps_req_payload), ps_req_payload, len(last_received_request_to_mocked_daemon), last_received_request_to_mocked_daemon)
		}
		resp_buf := make([]byte, 512)
		_, err = c.Read(resp_buf)
		if err != nil {
			return
		}
		resp_buf_str := string(resp_buf)
		log.Printf("Client -- Response received: %s\n", resp_buf_str)
		if resp_buf_str != last_sent_response_from_mocked_daemon {
			t.Errorf("Expected response (len %d):\n\n%s\n\nGot response (len %d):\n\n%s\n", len(last_sent_response_from_mocked_daemon), last_sent_response_from_mocked_daemon, len(resp_buf_str), resp_buf_str)
		}
	}
	log.Printf("====================================================================\n")
	log.Printf("====================================================================\n")
	// Also fire off a "docker run" API call.
	// "docker run -it --rm alpine:3.7 sh"
	run_req_payload := "POST /v1.37/containers/create HTTP/1.1\r\nHost: docker\r\nUser-Agent: Docker-Client/18.03.1-ce (darwin)\r\nContent-Length: 1426\r\nContent-Type: application/json\r\n\r\n"
	run_req_payload = fmt.Sprintf("%s%s", run_req_payload, `{"Hostname":"","Domainname":"","User":"","AttachStdin":true,"AttachStdout":true,"AttachStderr":true,"Tty":true,"OpenStdin":true,"StdinOnce":true,"Env":[],"Cmd":["sh"],"Image":"alpine:3.7","Volumes":{},"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":{},"HostConfig":{"Binds":null,"ContainerIDFile":"","LogConfig":{"Type":"","Config":{}},"NetworkMode":"default","PortBindings":{},"RestartPolicy":{"Name":"no","MaximumRetryCount":0},"AutoRemove":true,"VolumeDriver":"","VolumesFrom":null,"CapAdd":null,"CapDrop":null,"Dns":[],"DnsOptions":[],"DnsSearch":[],"ExtraHosts":null,"GroupAdd":null,"IpcMode":"","Cgroup":"","Links":null,"OomScoreAdj":0,"PidMode":"","Privileged":false,"PublishAllPorts":false,"ReadonlyRootfs":false,"SecurityOpt":null,"UTSMode":"","UsernsMode":"","ShmSize":0,"ConsoleSize":[0,0],"Isolation":"","CpuShares":0,"Memory":0,"NanoCpus":0,"CgroupParent":"","BlkioWeight":0,"BlkioWeightDevice":[],"BlkioDeviceReadBps":null,"BlkioDeviceWriteBps":null,"BlkioDeviceReadIOps":null,"BlkioDeviceWriteIOps":null,"CpuPeriod":0,"CpuQuota":0,"CpuRealtimePeriod":0,"CpuRealtimeRuntime":0,"CpusetCpus":"","CpusetMems":"","Devices":[],"DeviceCgroupRules":null,"DiskQuota":0,"KernelMemory":0,"MemoryReservation":0,"MemorySwap":0,"MemorySwappiness":-1,"OomKillDisable":false,"PidsLimit":0,"Ulimits":null,"CpuCount":0,"CpuPercent":0,"IOMaximumIOps":0,"IOMaximumBandwidth":0},"NetworkingConfig":{"EndpointsConfig":{}}}`)
	run_req_payload = fmt.Sprintf("%s%s", run_req_payload, "\r\n")
	log.Printf("Client -- Sending request: %s", run_req_payload)
	_, err = c.Write([]byte(run_req_payload))
	if err != nil {
		println(err.Error())
	}
	time.Sleep(100 * time.Millisecond)
	// TODO: this one needs a slightly differing payload validated, with the extra label/cgroup attached
	if run_req_payload != last_received_request_to_mocked_daemon {
		t.Errorf("Expected request (len %d):\n\n%s\n\nGot request (len %d):\n\n%s\n", len(run_req_payload), run_req_payload, len(last_received_request_to_mocked_daemon), last_received_request_to_mocked_daemon)
	}
	resp_buf := make([]byte, 512)
	_, err = c.Read(resp_buf)
	if err != nil {
		return
	}
	resp_buf_str := string(resp_buf)
	log.Printf("Client -- Response received: %s\n", resp_buf_str)
	if resp_buf_str != last_sent_response_from_mocked_daemon {
		t.Errorf("Expected response (len %d):\n\n%s\n\nGot response (len %d - buf size?):\n\n%s\n", len(last_sent_response_from_mocked_daemon), last_sent_response_from_mocked_daemon, len(resp_buf_str), resp_buf_str)
	}
	log.Printf("====================================================================\n")
	log.Printf("====================================================================\n")

	stopDockerProxy(&wg, &docker_proxy)
}
