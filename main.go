package main

import (
	"os"

	"gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "dockerd-ci-proxy"
	app.Usage = "Docker Daemon UNIX Socket Proxy for CI Child Containers"
	app.Version = dockerd_ci_proxy_version

	app.Run(os.Args)
}
