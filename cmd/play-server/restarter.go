package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

type Restart struct {
	ContainerId int
	ReadyCh     chan<- int
}

func Restarter(restarterId int, restartCh chan Restart, host string,
	containerPublishAddr string,
	containerPublishPortBase,
	containerPublishPortSpan int,
	portMapping [][]int) {
	StatsNumInc("Restarter")

	for restart := range restartCh {
		StatsNumInc("Restarter.loop.beg")

		start := time.Now()

		StatsNum("Restarter.containerId:max",
			func(cur uint64) uint64 {
				if uint64(restart.ContainerId) > cur {
					return uint64(restart.ContainerId)
				}
				return cur
			})

		cmd := exec.Command("make",
			fmt.Sprintf("CONTAINER_NUM=%d", restart.ContainerId))

		portBase := containerPublishPortBase +
			(containerPublishPortSpan * restart.ContainerId)

		ports := make([]string, 0, len(portMapping))
		for _, port := range portMapping {
			ports = append(ports, fmt.Sprintf("-p %s:%d:%d/tcp",
				containerPublishAddr, portBase+port[1], port[0]))
		}

		cmd.Args = append(cmd.Args, "SERVICE_HOST="+host)

		cmd.Args = append(cmd.Args,
			"CONTAINER_PORTS="+strings.Join(ports, " "))

		cmd.Args = append(cmd.Args, "restart")

		if restart.ContainerId > 0 {
			// Keep container #0 unpaused so that it
			// can still serve up the web login UI when
			// the session id isn't known yet.
			cmd.Args = append(cmd.Args, "instance-pause")
		}

		log.Printf("INFO: Restarter, restarterId: %d, containerId: %d\n",
			restarterId, restart.ContainerId)

		StatsNumInc("Restarter.restart")

		stdOutErr, err := cmd.CombinedOutput()
		if err != nil {
			StatsNumInc("Restarter.restart.err")

			log.Printf("ERROR: Restarter, restarterId: %d,"+
				" containerId: %d, cmd: %v, stdOutErr: %s, err: %v",
				restarterId, restart.ContainerId, cmd, stdOutErr, err)

			go func(restart Restart) {
				StatsNumInc("Restarter.restart.err.retry")

				restartCh <- restart // Async retry to restart again.

				StatsNumInc("Restarter.restart.err.retry.sent")
			}(restart)
		} else {
			StatsNumInc("Restarter.restart.ok")

			log.Printf("INFO: Restarter, restarterId: %d,"+
				" containerId: %d, took: %s\n",
				restarterId, restart.ContainerId, time.Since(start))

			restart.ReadyCh <- restart.ContainerId

			StatsNumInc("Restarter.restart.ok.sent")
		}

		StatsNumInc("Restarter.loop.end")
	}
}
