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
	DoneCh      chan<- int
}

func Restarter(restarterId int, restartCh chan Restart,
	containerPublishAddr string,
	containerPublishPortBase,
	containerPublishPortSpan int,
	portMapping [][]int) {
	StatsNumInc("tot.Restarter")

	for restart := range restartCh {
		StatsNumInc("tot.Restarter.loop.beg")

		start := time.Now()

		cmd := exec.Command("make",
			fmt.Sprintf("CONTAINER_NUM=%d", restart.ContainerId))

		portBase := containerPublishPortBase +
			(containerPublishPortSpan * restart.ContainerId)

		ports := make([]string, 0, len(portMapping))
		for _, port := range portMapping {
			ports = append(ports, fmt.Sprintf("-p %s:%d:%d/tcp",
				containerPublishAddr, portBase+port[1], port[0]))
		}

		cmd.Args = append(cmd.Args,
			"CONTAINER_PORTS="+strings.Join(ports, " "))

		cmd.Args = append(cmd.Args, "restart")

		log.Printf("INFO: Restarter, restarterId: %d, containerId: %d\n",
			restarterId, restart.ContainerId)

		StatsNumInc("tot.Restarter.restart")

		stdOutErr, err := cmd.CombinedOutput()
		if err != nil {
			StatsNumInc("tot.Restarter.restart.err")

			log.Printf("ERROR: Restarter, restarterId: %d,"+
				" containerId: %d, cmd: %v, stdOutErr: %s, err: %v",
				restarterId, restart.ContainerId, cmd, stdOutErr, err)

			go func(restart Restart) {
				StatsNumInc("tot.Restarter.restart.err.retry")

				restartCh <- restart // Async retry to restart again.

				StatsNumInc("tot.Restarter.restart.err.retry.sent")
			}(restart)
		} else {
			StatsNumInc("tot.Restarter.restart.ok")

			log.Printf("INFO: Restarter, restarterId: %d,"+
				" containerId: %d, took: %s\n",
				restarterId, restart.ContainerId, time.Since(start))

			restart.DoneCh <- restart.ContainerId

			StatsNumInc("tot.Restarter.restart.ok.sent")
		}

		StatsNumInc("tot.Restarter.loop.end")
	}
}
