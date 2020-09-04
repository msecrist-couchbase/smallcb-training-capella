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

func RunLangCode(ctx context.Context, user, execPrefix,
	lang, code string, codeDuration time.Duration,
	readyCh chan int,
	containerWaitDuration time.Duration,
	containerNamePrefix,
	containerVolPrefix string,
	restartCh chan<- Restart) ([]byte, error) {
	containerId, err := WaitForReadyContainer(ctx, readyCh, containerWaitDuration)
	if err != nil {
		return nil, err
	}

	defer func() {
		// If we didn't use the container (so it's clean),
		// we can immediately put the containerId token
		// back into the readyCh for the next client.
		if containerId >= 0 {
			readyCh <- containerId
		}
	}()

	result, err := RunLangCodeContainer(ctx, user,
		execPrefix, lang, code, codeDuration,
		containerId, containerNamePrefix, containerVolPrefix)

	go func(containerId int) {
		restartCh <- Restart{
			ContainerId: containerId,
			DoneCh:      readyCh,
		}
	}(containerId)

	// The restartCh now owns the containerId token.
	containerId = -1

	return result, err
}

// ------------------------------------------------

func RunLangCodeContainer(ctx context.Context, user, execPrefix,
	lang, code string, codeDuration time.Duration,
	containerId int,
	containerNamePrefix,
	containerVolPrefix string) ([]byte, error) {
	// Ex: "vol-instances/vol-0".
	dir := fmt.Sprintf("%s%d", containerVolPrefix, containerId)

	err := os.MkdirAll(dir+DirCode, 0777)
	if err != nil {
		return nil, err
	}

	// Ex: "vol-instances/vol-0/tmp/play/code.py".
	codePathHost := dir + DirCode + "/code." + lang

	// Ex: "/opt/couchbase/var/tmp/play/code.py".
	codePathInst := DirVar + DirCode + "/code." + lang

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
