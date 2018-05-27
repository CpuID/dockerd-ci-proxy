package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"github.com/CpuID/atexit"
	"gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "dockerd-ci-proxy"
	app.Usage = "Docker Daemon UNIX Socket Proxy for CI Child Containers"
	app.Version = dockerd_ci_proxy_version

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "Debug Mode",
		},
		cli.StringFlag{
			Name:  "dockersocket, ds",
			Value: "/var/run/docker.sock",
			Usage: "The Docker daemon API UNIX socket to connect to",
		},
		cli.StringFlag{
			Name:  "listensocket, ls",
			Value: "/var/run/docker-ci-proxy.sock",
			Usage: "The UNIX listen socket for this process, Docker API clients will point at this path",
		},
	}

	app.Action = func(c *cli.Context) error {
		// Trap SIGINT for Ctrl+C
		c_sig := make(chan os.Signal, 1)
		signal.Notify(c_sig, os.Interrupt)

		// Default before it's set correctly
		listen_socket_full_path := "NONE"
		use_listen_socket_full_path, err := filepath.Abs(c.String("listensocket"))
		if err != nil {
			log.Printf("Cannot determine full path of UNIX Listen Socket '%s', may be left orphaned.", c.String("listensocket"))
			atexit.Exit(1)
		}
		listen_socket_full_path = use_listen_socket_full_path

		// On exit, ensure the listen socket is deleted. Most reliable place for it is here
		atexit.Register(func() {
			if listen_socket_full_path != "NONE" {
				if _, err := os.Stat(listen_socket_full_path); err == nil {
					os.Remove(listen_socket_full_path)
					log.Printf("Closed UNIX socket '%s' deleted", listen_socket_full_path)
				}
			}
		})

		// Start the Docker Socket Proxy
		docker_proxy := dockerProxy{
			ListenSocket: c.String("listensocket"),
			TargetSocket: c.String("dockersocket"),
		}
		var wg sync.WaitGroup
		ready := make(chan int)
		wg.Add(1)
		go docker_proxy.runProxy(&wg, ready)

		// SIGINT handler
		go func() {
			// Block waiting for channel "c" to receive the signal.
			<-c_sig
			log.Println("Caught SIGINT, cleaning up...")
			log.Println("(closed channel warnings are safe to ignore here)")
			docker_proxy.StoppableListener.Stop()
			wg.Wait()
			atexit.Exit(2)
		}()

		// Channel notificatio comes in once the listen socket is ready to receive requests.
		<-ready
		log.Printf("Listening on '%s' for Docker API requests", listen_socket_full_path)

		// Sleep indefinitely, until a SIGINT signal is received
		for {
			time.Sleep(2 * time.Second)
		}

		return nil
	}

	app.Run(os.Args)
	atexit.Exit(0)
}
