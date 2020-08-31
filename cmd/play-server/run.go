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

func CheckLangCode(lang, code string, codeMaxLen int) (
	runnable bool, err error) {
	if lang == "" || code == "" {
		return false, nil
	}

	if len(code) > codeMaxLen {
		return false, fmt.Errorf("code too long, codeMaxLen: %d", codeMaxLen)
	}

	return true, nil
}

// ------------------------------------------------

func RunLangCode(ctx context.Context, user,
	execPrefix, lang, code string, codeDuration time.Duration,
	containersCh chan int,
	containerWaitDuration time.Duration,
	containerNamePrefix,
	containerVolPrefix string,
	restarterCh chan<- int) ([]byte, error) {
	// Atomically grab a containerId token, blocking
	// until a container instance is ready.
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
		return nil, fmt.Errorf("timeout waiting for container instance, duration: %v", containerWaitDuration)

	case <-ctx.Done():
		// Client canceled/timed-out while we were waiting.
		return nil, ctx.Err()
	}

	result, err := RunLangCodeContainer(ctx, user,
		execPrefix, lang, code, codeDuration,
		containerId, containerNamePrefix, containerVolPrefix)

	select {
	case restarterCh <- containerId:
		// The restarter now owns the containerId token.
		containerId = -1
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return result, err
}

// ------------------------------------------------

func RunLangCodeContainer(ctx context.Context, user,
	execPrefix, lang, code string, codeDuration time.Duration,
	containerId int,
	containerNamePrefix,
	containerVolPrefix string) ([]byte, error) {
	dir := fmt.Sprintf("%s%d", containerVolPrefix, containerId)

	err := os.MkdirAll(dir+"/tmp/play", 0777)
	if err != nil {
		return nil, err
	}

	// Ex: "vol-0/tmp/play/code.py".
	codePathHost := dir + "/tmp/play/code." + lang

	// Ex: "/opt/couchbase/var/tmp/play/code.py".
	codePathInst := "/opt/couchbase/var/tmp/play/code." + lang

	codeBytes := []byte(strings.ReplaceAll(code, "\r\n", "\n"))

	// File mode is 0777 executable, for scripts like 'code.py'.
	err = ioutil.WriteFile(codePathHost, codeBytes, 0777)
	if err != nil {
		return nil, err
	}

	// Ex: "smallcb-0".
	containerName := fmt.Sprintf("%s%d", containerNamePrefix, containerId)

	var cmd *exec.Cmd

	if len(execPrefix) > 0 {
		// Case of an execPrefix like "/run-java.sh".
		cmd = exec.Command("docker", "exec", "-u", user,
			containerName, execPrefix, codePathInst)
	} else {
		cmd = exec.Command("docker", "exec", "-u", user,
			containerName, codePathInst)
	}

	log.Printf("INFO: RunLangCodeContainer, containerId: %d, lang: %s\n",
		containerId, lang)

	return ExecCmd(ctx, cmd, codeDuration)
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

		log.Printf("INFO: Restarter, restarterId: %d, containerId: %d\n",
			restarterId, containerId)

		stdOutErr, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("ERROR: Restarter, restarterId: %d, containerId: %d,"+
				" cmd: %v, stdOutErr: %s, err: %v",
				restarterId, containerId, cmd, stdOutErr, err)

			// Async try to restart the containerId again.
			go func(containerId int) {
				needRestartCh <- containerId
			}(containerId)

			continue
		}

		log.Printf("INFO: Restarter, restarterId: %d, containerId: %d, took: %s\n",
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
