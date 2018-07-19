package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"testing"
	"time"
)

// last_received_request_to_mocked_daemon + last_sent_response_from_mocked_daemon set in main_test.go

func TestProxyDockerPs(t *testing.T) {
	// Fire off 2 x "docker ps" executions, validate standard passthrough behaviour
	for i := 0; i < 2; i++ {
		mocked_docker_daemon_mutex.Lock()
		ps_req_payload := "GET /v1.37/containers/json HTTP/1.1\r\nHost: docker\r\nUser-Agent: Docker-Client/18.03.1-ce (darwin)\r\n\r\n"
		if debug_mode >= 2 {
			log.Printf("====================================================================\n")
			log.Printf("====================================================================\n")
			log.Printf("Client -- Sending request: %s", ps_req_payload)
		}
		_, err := dc.Write([]byte(ps_req_payload))
		if err != nil {
			t.Fatal(err.Error())
		}
		time.Sleep(100 * time.Millisecond)
		if ps_req_payload != last_received_request_to_mocked_daemon {
			t.Errorf("Expected request (len %d):\n\n%s\n\nGot request (len %d):\n\n%s\n", len(ps_req_payload), ps_req_payload, len(last_received_request_to_mocked_daemon), last_received_request_to_mocked_daemon)
		}
		// TODOLATER: Use Content-Length header to determine EOF
		resp_buf := make([]byte, 512)
		_, err = dc.Read(resp_buf)
		if err != nil {
			return
		}
		resp_buf_str := string(bytes.TrimRight(resp_buf, "\x00"))
		if debug_mode >= 2 {
			log.Printf("Client -- Response received: %s\n", resp_buf_str)
		}
		if resp_buf_str != sortHeaders(last_sent_response_from_mocked_daemon) {
			t.Errorf("Expected response (len %d):\n\n%s\n\nGot response (len %d):\n\n%s\n", len(sortHeaders(last_sent_response_from_mocked_daemon)), sortHeaders(last_sent_response_from_mocked_daemon), len(resp_buf_str), resp_buf_str)
		}
		mocked_docker_daemon_mutex.Unlock()
	}
}

func TestProxyDockerRunLabel(t *testing.T) {
	mocked_docker_daemon_mutex.Lock()
	// Fire off a "docker run" API call.
	// "docker run -it --rm alpine:3.7 sh"
	run_req_payload := "POST /v1.37/containers/create HTTP/1.1\r\nHost: docker\r\nUser-Agent: Docker-Client/18.03.1-ce (darwin)\r\nContent-Length: 1426\r\nContent-Type: application/json\r\n\r\n"
	run_req_payload = fmt.Sprintf("%s%s", run_req_payload, `{"Hostname":"","Domainname":"","User":"","AttachStdin":true,"AttachStdout":true,"AttachStderr":true,"Tty":true,"OpenStdin":true,"StdinOnce":true,"Env":[],"Cmd":["sh"],"Image":"alpine:3.7","Volumes":{},"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":{},"HostConfig":{"Binds":null,"ContainerIDFile":"","LogConfig":{"Type":"","Config":{}},"NetworkMode":"default","PortBindings":{},"RestartPolicy":{"Name":"no","MaximumRetryCount":0},"AutoRemove":true,"VolumeDriver":"","VolumesFrom":null,"CapAdd":null,"CapDrop":null,"Dns":[],"DnsOptions":[],"DnsSearch":[],"ExtraHosts":null,"GroupAdd":null,"IpcMode":"","Cgroup":"","Links":null,"OomScoreAdj":0,"PidMode":"","Privileged":false,"PublishAllPorts":false,"ReadonlyRootfs":false,"SecurityOpt":null,"UTSMode":"","UsernsMode":"","ShmSize":0,"ConsoleSize":[0,0],"Isolation":"","CpuShares":0,"Memory":0,"NanoCpus":0,"CgroupParent":"","BlkioWeight":0,"BlkioWeightDevice":[],"BlkioDeviceReadBps":null,"BlkioDeviceWriteBps":null,"BlkioDeviceReadIOps":null,"BlkioDeviceWriteIOps":null,"CpuPeriod":0,"CpuQuota":0,"CpuRealtimePeriod":0,"CpuRealtimeRuntime":0,"CpusetCpus":"","CpusetMems":"","Devices":[],"DeviceCgroupRules":null,"DiskQuota":0,"KernelMemory":0,"MemoryReservation":0,"MemorySwap":0,"MemorySwappiness":-1,"OomKillDisable":false,"PidsLimit":0,"Ulimits":null,"CpuCount":0,"CpuPercent":0,"IOMaximumIOps":0,"IOMaximumBandwidth":0},"NetworkingConfig":{"EndpointsConfig":{}}}`)
	run_req_payload = fmt.Sprintf("%s%s", run_req_payload, "\r")
	if debug_mode >= 2 {
		log.Printf("====================================================================\n")
		log.Printf("====================================================================\n")
		log.Printf("Client -- Sending request: %s", run_req_payload)
	}
	_, err := dc.Write([]byte(run_req_payload))
	if err != nil {
		t.Fatal(err.Error())
	}
	time.Sleep(100 * time.Millisecond)
	// The modified HTTP request/payload received by the mock Docker daemon
	// Differences: Content-Type header, Labels
	expected_run_req_payload := "POST /v1.37/containers/create HTTP/1.1\r\nHost: docker\r\nUser-Agent: Docker-Client/18.03.1-ce (darwin)\r\nContent-Length: 1471\r\nContent-Type: application/json\r\n\r\n"
	expected_run_req_payload = fmt.Sprintf("%s%s", expected_run_req_payload, `{"Hostname":"","Domainname":"","User":"","AttachStdin":true,"AttachStdout":true,"AttachStderr":true,"Tty":true,"OpenStdin":true,"StdinOnce":true,"Env":[],"Cmd":["sh"],"Image":"alpine:3.7","Volumes":{},"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":{"Created-Via-Mock-By":"dockerd-ci-proxy-test"},"HostConfig":{"Binds":null,"ContainerIDFile":"","LogConfig":{"Type":"","Config":{}},"NetworkMode":"default","PortBindings":{},"RestartPolicy":{"Name":"no","MaximumRetryCount":0},"AutoRemove":true,"VolumeDriver":"","VolumesFrom":null,"CapAdd":null,"CapDrop":null,"Dns":[],"DnsOptions":[],"DnsSearch":[],"ExtraHosts":null,"GroupAdd":null,"IpcMode":"","Cgroup":"","Links":null,"OomScoreAdj":0,"PidMode":"","Privileged":false,"PublishAllPorts":false,"ReadonlyRootfs":false,"SecurityOpt":null,"UTSMode":"","UsernsMode":"","ShmSize":0,"ConsoleSize":[0,0],"Isolation":"","CpuShares":0,"Memory":0,"NanoCpus":0,"CgroupParent":"","BlkioWeight":0,"BlkioWeightDevice":[],"BlkioDeviceReadBps":null,"BlkioDeviceWriteBps":null,"BlkioDeviceReadIOps":null,"BlkioDeviceWriteIOps":null,"CpuPeriod":0,"CpuQuota":0,"CpuRealtimePeriod":0,"CpuRealtimeRuntime":0,"CpusetCpus":"","CpusetMems":"","Devices":[],"DeviceCgroupRules":null,"DiskQuota":0,"KernelMemory":0,"MemoryReservation":0,"MemorySwap":0,"MemorySwappiness":-1,"OomKillDisable":false,"PidsLimit":0,"Ulimits":null,"CpuCount":0,"CpuPercent":0,"IOMaximumIOps":0,"IOMaximumBandwidth":0},"NetworkingConfig":{"EndpointsConfig":{}}}`)
	expected_run_req_payload = fmt.Sprintf("%s%s", expected_run_req_payload, "\r")
	if expected_run_req_payload != last_received_request_to_mocked_daemon {
		t.Errorf("Expected request (len %d):\n\n%s\n\nGot request (len %d):\n\n%s\n", len(expected_run_req_payload), expected_run_req_payload, len(last_received_request_to_mocked_daemon), last_received_request_to_mocked_daemon)
	}
	// TODOLATER: Use Content-Length header to determine EOF
	resp_buf := make([]byte, 512)
	_, err = dc.Read(resp_buf)
	if err != nil {
		return
	}
	resp_buf_str := string(bytes.TrimRight(resp_buf, "\x00"))
	if debug_mode >= 2 {
		log.Printf("Client -- Response received: %s\n", resp_buf_str)
	}
	if resp_buf_str != sortHeaders(last_sent_response_from_mocked_daemon) {
		t.Errorf("Expected response (len %d):\n\n%s\n\nGot response (len %d - buf size?):\n\n%s\n", len(sortHeaders(last_sent_response_from_mocked_daemon)), sortHeaders(last_sent_response_from_mocked_daemon), len(resp_buf_str), resp_buf_str)
	}
	if debug_mode >= 2 {
		log.Printf("====================================================================\n")
		log.Printf("====================================================================\n")
	}
	mocked_docker_daemon_mutex.Unlock()
}

func TestProxyDockerRunLabelAndCgroupParent(t *testing.T) {
	mocked_docker_daemon_mutex.Lock()
	// Fire off a "docker run" API call.
	// "docker run -it --rm alpine:3.7 sh"
	run_req_payload := "POST /v1.37/containers/create HTTP/1.1\r\nHost: docker\r\nUser-Agent: Docker-Client/18.03.1-ce (darwin)\r\nContent-Length: 1426\r\nContent-Type: application/json\r\n\r\n"
	run_req_payload = fmt.Sprintf("%s%s", run_req_payload, `{"Hostname":"","Domainname":"","User":"","AttachStdin":true,"AttachStdout":true,"AttachStderr":true,"Tty":true,"OpenStdin":true,"StdinOnce":true,"Env":[],"Cmd":["sh"],"Image":"alpine:3.7","Volumes":{},"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":{},"HostConfig":{"Binds":null,"ContainerIDFile":"","LogConfig":{"Type":"","Config":{}},"NetworkMode":"default","PortBindings":{},"RestartPolicy":{"Name":"no","MaximumRetryCount":0},"AutoRemove":true,"VolumeDriver":"","VolumesFrom":null,"CapAdd":null,"CapDrop":null,"Dns":[],"DnsOptions":[],"DnsSearch":[],"ExtraHosts":null,"GroupAdd":null,"IpcMode":"","Cgroup":"","Links":null,"OomScoreAdj":0,"PidMode":"","Privileged":false,"PublishAllPorts":false,"ReadonlyRootfs":false,"SecurityOpt":null,"UTSMode":"","UsernsMode":"","ShmSize":0,"ConsoleSize":[0,0],"Isolation":"","CpuShares":0,"Memory":0,"NanoCpus":0,"CgroupParent":"","BlkioWeight":0,"BlkioWeightDevice":[],"BlkioDeviceReadBps":null,"BlkioDeviceWriteBps":null,"BlkioDeviceReadIOps":null,"BlkioDeviceWriteIOps":null,"CpuPeriod":0,"CpuQuota":0,"CpuRealtimePeriod":0,"CpuRealtimeRuntime":0,"CpusetCpus":"","CpusetMems":"","Devices":[],"DeviceCgroupRules":null,"DiskQuota":0,"KernelMemory":0,"MemoryReservation":0,"MemorySwap":0,"MemorySwappiness":-1,"OomKillDisable":false,"PidsLimit":0,"Ulimits":null,"CpuCount":0,"CpuPercent":0,"IOMaximumIOps":0,"IOMaximumBandwidth":0},"NetworkingConfig":{"EndpointsConfig":{}}}`)
	run_req_payload = fmt.Sprintf("%s%s", run_req_payload, "\r")
	if debug_mode >= 2 {
		log.Printf("====================================================================\n")
		log.Printf("====================================================================\n")
		log.Printf("Client -- Sending request: %s", run_req_payload)
	}
	// Enable Cgroup Parent support
	docker_cgroup_parent = "asdf"
	_, err := dc.Write([]byte(run_req_payload))
	if err != nil {
		t.Fatal(err.Error())
	}
	time.Sleep(100 * time.Millisecond)
	// The modified HTTP request/payload received by the mock Docker daemon
	// Differences: Content-Type header, Labels
	expected_run_req_payload := "POST /v1.37/containers/create HTTP/1.1\r\nHost: docker\r\nUser-Agent: Docker-Client/18.03.1-ce (darwin)\r\nContent-Length: 1471\r\nContent-Type: application/json\r\n\r\n"
	expected_run_req_payload = fmt.Sprintf("%s%s", expected_run_req_payload, `{"Hostname":"","Domainname":"","User":"","AttachStdin":true,"AttachStdout":true,"AttachStderr":true,"Tty":true,"OpenStdin":true,"StdinOnce":true,"Env":[],"Cmd":["sh"],"Image":"alpine:3.7","Volumes":{},"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":{"Created-Via-Mock-By":"dockerd-ci-proxy-test"},"HostConfig":{"Binds":null,"ContainerIDFile":"","LogConfig":{"Type":"","Config":{}},"NetworkMode":"default","PortBindings":{},"RestartPolicy":{"Name":"no","MaximumRetryCount":0},"AutoRemove":true,"VolumeDriver":"","VolumesFrom":null,"CapAdd":null,"CapDrop":null,"Dns":[],"DnsOptions":[],"DnsSearch":[],"ExtraHosts":null,"GroupAdd":null,"IpcMode":"","Cgroup":"","Links":null,"OomScoreAdj":0,"PidMode":"","Privileged":false,"PublishAllPorts":false,"ReadonlyRootfs":false,"SecurityOpt":null,"UTSMode":"","UsernsMode":"","ShmSize":0,"ConsoleSize":[0,0],"Isolation":"","CpuShares":0,"Memory":0,"NanoCpus":0,"CgroupParent":"asdf","BlkioWeight":0,"BlkioWeightDevice":[],"BlkioDeviceReadBps":null,"BlkioDeviceWriteBps":null,"BlkioDeviceReadIOps":null,"BlkioDeviceWriteIOps":null,"CpuPeriod":0,"CpuQuota":0,"CpuRealtimePeriod":0,"CpuRealtimeRuntime":0,"CpusetCpus":"","CpusetMems":"","Devices":[],"DeviceCgroupRules":null,"DiskQuota":0,"KernelMemory":0,"MemoryReservation":0,"MemorySwap":0,"MemorySwappiness":-1,"OomKillDisable":false,"PidsLimit":0,"Ulimits":null,"CpuCount":0,"CpuPercent":0,"IOMaximumIOps":0,"IOMaximumBandwidth":0},"NetworkingConfig":{"EndpointsConfig":{}}}`)
	expected_run_req_payload = fmt.Sprintf("%s%s", expected_run_req_payload, "\r")
	if expected_run_req_payload != last_received_request_to_mocked_daemon {
		t.Errorf("Expected request (len %d):\n\n%s\n\nGot request (len %d):\n\n%s\n", len(expected_run_req_payload), expected_run_req_payload, len(last_received_request_to_mocked_daemon), last_received_request_to_mocked_daemon)
	}
	// TODOLATER: Use Content-Length header to determine EOF
	resp_buf := make([]byte, 512)
	_, err = dc.Read(resp_buf)
	if err != nil {
		return
	}
	resp_buf_str := string(bytes.TrimRight(resp_buf, "\x00"))
	if debug_mode >= 2 {
		log.Printf("Client -- Response received: %s\n", resp_buf_str)
	}
	if resp_buf_str != sortHeaders(last_sent_response_from_mocked_daemon) {
		t.Errorf("Expected response (len %d):\n\n%s\n\nGot response (len %d - buf size?):\n\n%s\n", len(sortHeaders(last_sent_response_from_mocked_daemon)), sortHeaders(last_sent_response_from_mocked_daemon), len(resp_buf_str), resp_buf_str)
	}
	if debug_mode >= 2 {
		log.Printf("====================================================================\n")
		log.Printf("====================================================================\n")
	}
	// Disable Cgroup Parent support
	docker_cgroup_parent = ""
	mocked_docker_daemon_mutex.Unlock()
}

// TODOLATER: test an API call with query params, ensure URL/Path are handled as designed in MITM code
// (verified manually in import test below using debug output, currently working)

func TestProxyDockerImport(t *testing.T) {
	// Skip for now, broken implementation? Mock docker daemon doesn't give up trying to read the HTTP request.
	t.Skip("Disabled")

	mocked_docker_daemon_mutex.Lock()
	// Also fire off a "docker import" API call.
	// "docker import fixtures/layer.tar"
	import_req_payload := []byte("POST /v1.31/images/create?fromSrc=-&message=&repo=&tag= HTTP/1.1\r\nHost: docker\r\nUser-Agent: Docker-Client/17.07.0-ce (linux)\r\nTransfer-Encoding: chunked\r\nContent-Type: text/plain\r\n\r\n")
	fixtures_layer_tar, err := ioutil.ReadFile("./fixtures/layer.tar")
	if err != nil {
		t.Fatal(err.Error())
	}
	trimmed_fixtures_layer_tar := bytes.Trim(fixtures_layer_tar, "\x00")
	import_req_payload = append(import_req_payload, []byte("2800\r\n")...)

	// Mocked payload for now, real payload seemed buggy?
	//trimmed_fixtures_layer_tar := []byte("aaaabbbbcccc")
	//import_req_payload = append(import_req_payload, []byte("12\r\n")...)
	//
	import_req_payload = append(import_req_payload, trimmed_fixtures_layer_tar...)
	import_req_payload = append(import_req_payload, []byte("\r\n0\r\n\r\n")...)
	if debug_mode >= 2 {
		log.Printf("====================================================================\n")
		log.Printf("====================================================================\n")
		log.Printf("Client -- (Trimmed) layer.tar size in bytes: %d\n", len(trimmed_fixtures_layer_tar))
		log.Printf("Client -- Sending request: %s", import_req_payload)
	}
	_, err = dc.Write(import_req_payload)
	if err != nil {
		t.Fatal(err.Error())
	}
	time.Sleep(100 * time.Millisecond)
	// No payload modification on "docker import", as labels cannot be specified (not ideal in reality, but that's the way it is)
	// TODOLATER: If we *really* wanted to be tricky here, the "docker import" CLI supposedly supports -c which runs a Dockerfile instruction
	// against the imported image. This could be "LABEL name=value" and fulfil our goal.
	expected_import_req_payload := import_req_payload
	if string(expected_import_req_payload) != last_received_request_to_mocked_daemon {
		t.Errorf("Expected request (len %d):\n\n%s\n\nGot request (len %d):\n\n%s\n", len(expected_import_req_payload), expected_import_req_payload, len(last_received_request_to_mocked_daemon), last_received_request_to_mocked_daemon)
	}
	// TODOLATER: Use Content-Length header to determine EOF
	resp_buf := make([]byte, 512)
	_, err = dc.Read(resp_buf)
	if err != nil {
		return
	}
	resp_buf_str := string(bytes.TrimRight(resp_buf, "\x00"))
	if debug_mode >= 2 {
		log.Printf("Client -- Response received: %s\n", resp_buf_str)
	}
	if resp_buf_str != sortHeaders(last_sent_response_from_mocked_daemon) {
		t.Errorf("Expected response (len %d):\n\n%s\n\nGot response (len %d - buf size?):\n\n%s\n", len(sortHeaders(last_sent_response_from_mocked_daemon)), sortHeaders(last_sent_response_from_mocked_daemon), len(resp_buf_str), resp_buf_str)
	}
	if debug_mode >= 2 {
		log.Printf("====================================================================\n")
		log.Printf("====================================================================\n")
	}
	mocked_docker_daemon_mutex.Unlock()
}
