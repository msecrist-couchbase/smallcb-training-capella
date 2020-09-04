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
	StatsNumInc("Restarter tot")

	for restart := range restartCh {
		StatsNumInc("Restarter.loop.beg tot")

		start := time.Now()

		StatsNum("Restarter.containerId max",
			func(cur uint64) uint64 {
				if uint64(restart.ContainerId) > cur {
					return uint64(restart.ContainerId)
				}
				return cur
			})

		StatsNum("Restarter.containerId min",
			func(cur uint64) uint64 {
				if uint64(restart.ContainerId) < cur {
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

		cmd.Args = append(cmd.Args,
			"CONTAINER_PORTS="+strings.Join(ports, " "))

		cmd.Args = append(cmd.Args, "restart")

		log.Printf("INFO: Restarter, restarterId: %d, containerId: %d\n",
			restarterId, restart.ContainerId)

		StatsNumInc("Restarter.restart tot")

		stdOutErr, err := cmd.CombinedOutput()
		if err != nil {
			StatsNumInc("Restarter.restart.err tot")

			log.Printf("ERROR: Restarter, restarterId: %d,"+
				" containerId: %d, cmd: %v, stdOutErr: %s, err: %v",
				restarterId, restart.ContainerId, cmd, stdOutErr, err)

			go func(restart Restart) {
				StatsNumInc("Restarter.restart.err.retry tot")

				restartCh <- restart // Async retry to restart again.

				StatsNumInc("Restarter.restart.err.retry.sent tot")
			}(restart)
		} else {
			StatsNumInc("Restarter.restart.ok tot")

			log.Printf("INFO: Restarter, restarterId: %d,"+
				" containerId: %d, took: %s\n",
				restarterId, restart.ContainerId, time.Since(start))

			restart.DoneCh <- restart.ContainerId

			StatsNumInc("Restarter.restart.ok.sent tot")
		}

		StatsNumInc("Restarter.loop.end tot")
	}
}
