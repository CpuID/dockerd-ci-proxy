package main

import (
	"io"
	"log"
	"net"
	"sync"
)

// Credit: http://blog.csdn.net/ylqmf/article/details/38856179

// TODO: adjust verbosity further down once tested.

type dockerProxy struct {
	ListenSocket      string
	TargetSocket      string
	StoppableListener *stoppableUnixListener
}

func (s *dockerProxy) runProxy(wg *sync.WaitGroup, ready chan<- int) {
	l, err := net.Listen("unix", s.ListenSocket)
	if err != nil {
		log.Fatalf("Docker Label Proxy - Initial UNIX Listen Error: %s\n", err.Error())
	}
	sl, err := newStoppableUnixListener(l)
	if err != nil {
		log.Fatalf("Docker Label Proxy - Stoppable UNIX Listen Error: %s\n", err.Error())
	}
	s.StoppableListener = sl

	defer wg.Done()
	first := true
	for {
		if first {
			// Notify parent that we are ready.
			ready <- 1
			first = false
		}
		tc, err := s.StoppableListener.Accept()
		if err != nil {
			// Check if it is due to the listener being stopped, or some other reason.
			if err.Error() == "Listener Stopped" {
				// Stop channel triggered, unroll our loops.
				break
			} else {
				log.Fatalf("Accept UNIX Conn Error: %s\n", err.Error())
			}
		}

		go s.eachConn(tc)
	}
}

func (s *dockerProxy) eachConn(tc net.Conn) {
	uc, err := net.Dial("unix", s.TargetSocket)
	if err != nil {
		log.Printf("Docker Label Proxy - Failed to connect to UNIX Socket %s. Error: %s\n", s.ListenSocket, err.Error())
		uc.Close()
		return
	}

	// Passthrough requests to Docker daemon, testing use only
	// go io.Copy(tc, uc)
	// Intercept/MITM the requests to Docker daemon
	go mitmDockerApiCall(tc, uc)

	// Response is propagated unmodified from upstream Docker socket to client
	go io.Copy(uc, tc)
}

func startDockerProxy(proxy_wg *sync.WaitGroup, docker_proxy *dockerProxy, proxy_ready chan int, target_socket string, listen_socket string) {
	log.Printf("Starting Docker Label Proxy (Listening on %s)... \n", listen_socket)
	docker_proxy.ListenSocket = listen_socket
	docker_proxy.TargetSocket = target_socket
	proxy_wg.Add(1)
	go docker_proxy.runProxy(proxy_wg, proxy_ready)
	<-proxy_ready
	log.Println("Started.")
}

func stopDockerProxy(proxy_wg *sync.WaitGroup, docker_proxy *dockerProxy) {
	log.Println("Stopping Docker Label Proxy...")
	docker_proxy.StoppableListener.Stop()
	proxy_wg.Wait()
	log.Println("Docker Label Proxy stopped.")
}

// Usage:
//
// docker_proxy := dockerProxy{ListenSocket: "/var/run/docker_proxy.sock", TargetSocket: "/var/run/docker.sock"}
// var wg sync.WaitGroup
// ready := make(chan int)
// wg.Add(1)
// go docker_proxy.runProxy(&wg, ready)
// <-ready
// Do some stuff...
// docker_proxy.StoppableListener.Stop()
// wg.Wait()
