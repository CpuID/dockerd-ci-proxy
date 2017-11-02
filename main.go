package main

import (
	"os"

	"gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "dockerd-label-proxy"
	app.Usage = "Docker Daemon UNIX Socket Label Proxy"
	app.Version = dockerd_label_proxy_version

	app.Run(os.Args)
}
