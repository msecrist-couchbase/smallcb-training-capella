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

var (
	DirVar  = "/opt/couchbase/var"
	DirCode = "/tmp/play"
)

func CheckLangCode(lang, code string, codeMaxLen int) (
	runnable bool, err error) {
	if code == "" {
		return false, nil
	}

	if len(code) > codeMaxLen {
		return false, fmt.Errorf("code too long, codeMaxLen: %d", codeMaxLen)
	}

	return true, nil
}

// ------------------------------------------------

// RunRequestSingle waits for a ready container instance,
// runs code once (just a single request, not a session),
// then asynchronously restarts that container instance.
func RunRequestSingle(req RunRequest, readyCh chan int,
	containerWaitDuration time.Duration,
	restartCh chan<- Restart) ([]byte, error) {
	containerId, err := WaitForReadyContainer(
		req.ctx, readyCh, containerWaitDuration)
	if err != nil {
		return nil, err
	}

	defer func() {
		// If we didn't use the container, we
		// put the containerId token back into
		// the readyCh for the next client's use.
		if containerId >= 0 {
			readyCh <- containerId
		}
	}()

	result, err := RunRequestInContainer(req, containerId)

	go func(containerId int) {
		restartCh <- Restart{
			ContainerId: containerId,
			ReadyCh:     readyCh,
		}
	}(containerId)

	// The restartCh now owns the containerId token.
	containerId = -1

	return result, err
}

// ------------------------------------------------

type RunRequest struct {
	ctx context.Context

	execUser     string
	execPrefix   string
	lang         string
	code         string
	codeDuration time.Duration

	containerNamePrefix string
	containerVolPrefix  string
}

func RunRequestInContainer(req RunRequest, containerId int) (
	[]byte, error) {
	// Ex: "vol-instances/vol-0".
	dir := fmt.Sprintf("%s%d",
		req.containerVolPrefix, containerId)

	err := os.MkdirAll(dir+DirCode, 0777)
	if err != nil {
		return nil, err
	}

	// Ex: "vol-instances/vol-0/tmp/play/code.py".
	codePathHost := dir + DirCode + "/code." + req.lang

	// Ex: "/opt/couchbase/var/tmp/play/code.py".
	codePathInst := DirVar + DirCode + "/code." + req.lang

	codeBytes := []byte(strings.ReplaceAll(
		req.code, "\r\n", "\n"))

	// File mode 0777 executable, for scripts like 'code.py'.
	err = ioutil.WriteFile(codePathHost, codeBytes, 0777)
	if err != nil {
		return nil, err
	}

	// Ex: "smallcb-0".
	containerName := fmt.Sprintf("%s%d",
		req.containerNamePrefix, containerId)

	var cmd *exec.Cmd

	if len(req.execPrefix) > 0 {
		// Case of an execPrefix like "/run-java.sh".
		cmd = exec.Command("docker", "exec",
			"-u", req.execUser, containerName,
			req.execPrefix, codePathInst)
	} else {
		cmd = exec.Command("docker", "exec",
			"-u", req.execUser, containerName,
			codePathInst)
	}

	log.Printf("INFO: RunRequest, containerId: %d,"+
		" req.lang: %s\n", containerId, req.lang)

	return ExecCmd(req.ctx, cmd, req.codeDuration)
}

// ------------------------------------------------

// Run a cmd, waiting for it to finish or timeout,
// returning its combined stdout and stderr result.
func ExecCmd(ctx context.Context, cmd *exec.Cmd,
	duration time.Duration) ([]byte, error) {
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

// ------------------------------------------------

func WaitForReadyContainer(ctx context.Context, readyCh chan int,
	containerWaitDuration time.Duration) (int, error) {
	StatsNumInc("WaitForReadyContainer")

	select {
	case containerId := <-readyCh:
		StatsNumInc("WaitForReadyContainer.ready")

		return containerId, nil

	case <-time.After(containerWaitDuration):
		StatsNumInc("WaitForReadyContainer.timeout")

		return -1, fmt.Errorf("timeout waiting for container instance,"+
			" duration: %v", containerWaitDuration)

	case <-ctx.Done():
		StatsNumInc("WaitForReadyContainer.done")

		return -1, ctx.Err() // Client canceled/timed-out.
	}
}
