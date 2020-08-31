package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func RunLangCode(ctx context.Context, execPrefix,
	user, lang, code string,
	codeMaxLen int, codeDuration time.Duration,
	containersCh chan int,
	containerWaitDuration time.Duration,
	containerNamePrefix,
	containerVolPrefix string,
	restarterCh chan<- int) (
	string, error) {
	if lang == "" || code == "" {
		return "", nil
	}

	if len(code) > codeMaxLen {
		return "", fmt.Errorf("code too long, codeMaxLen: %d", codeMaxLen)
	}

	// Atomically grab a containerId token, blocking & waiting
	// until a container instance is available.
	var containerId int

	select {
	case containerId = <-containersCh:
		defer func() {
			// Put the token back for the next request
			// handler if we still have it.
			if containerId >= 0 {
				containersCh <- containerId
			}
		}()

	case <-time.After(containerWaitDuration):
		return "", fmt.Errorf("timeout waiting for worker, duration: %v", containerWaitDuration)

	case <-ctx.Done():
		// Client canceled/timed-out while we were waiting.
		return "", ctx.Err()
	}

	// A worker is ready & assigned, so prepare the code dir & file.
	dir := fmt.Sprintf("%s%d", containerVolPrefix, containerId)

	err := os.MkdirAll(dir+"/tmp/play", 0777)
	if err != nil {
		return "", err
	}

	// Ex: "vol-0/tmp/play/code.py".
	codePathHost := dir + "/tmp/play/code." + lang

	// Ex: "/opt/couchbase/var/tmp/play/code.py".
	codePathInst := "/opt/couchbase/var/tmp/play/code." + lang

	codeBytes := []byte(strings.ReplaceAll(code, "\r\n", "\n"))

	// Mode is 0777 executable, for scripts like 'code.py'.
	err = ioutil.WriteFile(codePathHost, codeBytes, 0777)
	if err != nil {
		return "", err
	}

	// Ex: "smallcb-0".
	containerName := fmt.Sprintf("%s%d", containerNamePrefix, containerId)

	var cmd *exec.Cmd

	if len(execPrefix) > 0 {
		// Case of an execPrefix like "/run-java.sh".
		cmd = exec.Command("docker", "exec",
			"-u", user,
			containerName, execPrefix, codePathInst)
	} else {
		cmd = exec.Command("docker", "exec",
			"-u", user,
			containerName, codePathInst)
	}

	log.Printf("INFO: running cmd, containerName: %s, codePathInst: %s\n",
		containerName, codePathInst)

	stdOutErr, err := ExecCmd(ctx, cmd, codeDuration)

	select {
	case restarterCh <- containerId:
		// The restarter now owns the containerId token.
		containerId = -1
	case <-ctx.Done():
		return "", nil
	}

	return string(stdOutErr), err
}

// ------------------------------------------------

func Restarter(restarterId int, needRestartCh, doneRestartCh chan int,
	containerPublishAddr string,
	containerPublishPortBase,
	containerPublishPortSpan int,
	portMapping [][]int) {
	for containerId := range needRestartCh {
		start := time.Now()

		cmd := exec.Command("make",
			fmt.Sprintf("CONTAINER_NUM=%d", containerId))

		portBase := containerPublishPortBase + (containerPublishPortSpan * containerId)

		ports := make([]string, 0, len(portMapping))
		for _, port := range portMapping {
			ports = append(ports,
				fmt.Sprintf("-p %s:%d:%d/tcp",
					containerPublishAddr, portBase+port[1], port[0]))
		}

		cmd.Args = append(cmd.Args,
			"CONTAINER_PORTS="+strings.Join(ports, " "))

		cmd.Args = append(cmd.Args, "restart")

		log.Printf("INFO: restarterId: %d, containerId: %d\n",
			restarterId, containerId)

		stdOutErr, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("ERROR: restarterId: %d, containerId: %d,"+
				" cmd: %v, stdOutErr: %s, err: %v",
				restarterId, containerId, cmd, stdOutErr, err)

			// Async try to restart the containerId again.
			go func(containerId int) {
				needRestartCh <- containerId
			}(containerId)

			continue
		}

		log.Printf("INFO: restarterId: %d, containerId: %d, took: %s\n",
			restarterId, containerId, time.Since(start))

		doneRestartCh <- containerId
	}
}

// ------------------------------------------------

// Run a cmd, waiting for it to finish or timeout,
// returning its combined stdout and stderr result.
func ExecCmd(ctx context.Context, cmd *exec.Cmd, duration time.Duration) (
	[]byte, error) {
	var b bytes.Buffer

	cmd.Stdout = &b
	cmd.Stderr = &b

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cmd.Start, err: %v", err)
	}

	doneCh := make(chan error, 1)
	go func() {
		doneCh <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		cmd.Process.Kill()
		return nil, fmt.Errorf("ctx.Done, err: %v", ctx.Err())

	case <-time.After(duration):
		cmd.Process.Kill()
		return nil, fmt.Errorf("timeout, duration: %v", duration)

	case err := <-doneCh:
		if err != nil {
			return nil, fmt.Errorf("doneCh, err: %v", err)
		}
	}

	return b.Bytes(), nil
}
