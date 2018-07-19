package main

import (
	"fmt"
	"log"

	"gopkg.in/urfave/cli.v1"
)

// TODOLATER: don't use globals here?
// 0 = none, 1 = some, 2 = all
var debug_mode uint8
var docker_cgroup_parent = ""
var docker_label_name = ""
var docker_label_value = ""

func handleFlags(c *cli.Context) error {
	if c.Bool("debug") == true {
		// "some" is enough here, "all" is mostly used in test coverage
		debug_mode = 1
	}

	if c.String("dockersocket") == "" {
		return fmt.Errorf("--dockersocket (--ds, or env DCP_DOCKER_SOCKET) is empty")
	}
	// TODOLATER: do we verify this socket exists here?
	if c.String("listensocket") == "" {
		return fmt.Errorf("--listensocket (--ls, or env DCP_LISTEN_SOCKET) is empty")
	}

	if c.Bool("cgroupparent") == true {
		d_cgp_result, err := thisContainerCgroupParent(c.String("dockersocket"))
		if err != nil {
			return err
		}
		// Flag enabled and non-empty on this container, set it.
		docker_cgroup_parent = d_cgp_result
		log.Printf("CgroupParent = '%s', enabled.", docker_cgroup_parent)
	}

	if c.String("labelname") == "" {
		return fmt.Errorf("--labelname (--ln, or env DCP_LABEL_NAME) is empty")
	}
	docker_label_name = c.String("labelname")
	if c.String("labelvalue") == "" {
		return fmt.Errorf("--labelvalue (--lv, or env DCP_LABEL_VALUE) is empty")
	}
	docker_label_value = c.String("labelvalue")

	return nil
}
