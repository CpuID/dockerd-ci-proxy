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

var app_code_name = "dockerd-ci-proxy"
var app_general_name = "Docker CI Proxy"

func main() {
	app := cli.NewApp()
	app.Name = app_code_name
	app.Usage = "Docker Daemon UNIX Socket Proxy for CI Child Containers"
	app.Version = dockerd_ci_proxy_version

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "debug, d",
			EnvVar: "DCP_DEBUG",
			Usage:  "Debug Mode",
		},
		cli.StringFlag{
			Name:   "dockersocket, ds",
			EnvVar: "DCP_DOCKER_SOCKET",
			Value:  "/var/run/docker.sock",
			Usage:  "The Docker daemon API UNIX socket to connect to",
		},
		cli.StringFlag{
			Name:   "listensocket, ls",
			EnvVar: "DCP_LISTEN_SOCKET",
			Value:  "/var/run/docker-ci-proxy.sock",
			Usage:  "The UNIX listen socket for this process, Docker API clients will point at this path",
		},
		cli.BoolFlag{
			Name:   "cgroupparent, cp",
			EnvVar: "DCP_CGROUP_PARENT",
			Usage:  "If enabled, overrides the CgroupParent of create operations to match this container",
		},
		cli.StringFlag{
			Name:   "labelname, ln",
			EnvVar: "DCP_LABEL_NAME",
			Value:  "Created-Via",
			Usage:  "The Docker label name to apply to resources",
		},
		cli.StringFlag{
			Name:   "labelvalue, lv",
			EnvVar: "DCP_LABEL_VALUE",
			Value:  "dockerd-ci-proxy",
			Usage:  "The Docker label value to apply to resources",
		},
	}

	app.Action = func(c *cli.Context) error {
		// Trap SIGINT for Ctrl+C
		c_sig := make(chan os.Signal, 1)
		signal.Notify(c_sig, os.Interrupt)

		err := handleFlags(c)
		if err != nil {
			log.Printf("%s -=- Startup failure: %s\n", app_code_name, err.Error())
			atexit.Exit(1)
		}

		// Sanitise listen socket path
		listen_socket_full_path, err := filepath.Abs(c.String("listensocket"))
		if err != nil {
			log.Printf("%s -=- Cannot determine full path of UNIX Listen Socket '%s', may be left orphaned.", app_code_name, c.String("listensocket"))
			atexit.Exit(1)
		}

		// On exit, ensure the listen socket is deleted. Most reliable place for it is here
		atexit.Register(func() {
			if _, err := os.Stat(listen_socket_full_path); err == nil {
				os.Remove(listen_socket_full_path)
				log.Printf("%s -=- Closed UNIX socket '%s' deleted", app_code_name, listen_socket_full_path)
			}
		})

		// Start the Docker Socket Proxy
		docker_proxy := dockerProxy{}
		var wg sync.WaitGroup
		ready := make(chan int)
		startDockerProxy(&wg, &docker_proxy, ready, c.String("dockersocket"), c.String("listensocket"))
		defer os.Remove(c.String("listensocket"))

		// SIGINT handler
		go func() {
			// Block waiting for channel "c" to receive the signal.
			<-c_sig
			log.Println("Caught SIGINT, cleaning up...")
			log.Println("(closed channel warnings are safe to ignore here)")
			// Stop the proxy
			stopDockerProxy(&wg, &docker_proxy)
			atexit.Exit(2)
		}()

		// Channel notification comes in once the listen socket is ready to receive requests.
		<-ready
		log.Printf("%s -=- Listening on '%s' for Docker API requests", app_code_name, listen_socket_full_path)

		// Sleep indefinitely, until a SIGINT signal is received
		for {
			time.Sleep(2 * time.Second)
		}

		return nil
	}

	app.Run(os.Args)
	atexit.Exit(0)
}
