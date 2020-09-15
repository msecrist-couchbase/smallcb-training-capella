package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
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
		return false, fmt.Errorf("code too long,"+
			" codeMaxLen: %d", codeMaxLen)
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
		go func() {
			restartCh <- Restart{
				ContainerId: containerId,
				ReadyCh:     readyCh,
			}
		}()
	}()

	err = AddRBACUser(req, containerId,
		"username", "password", "admin")
	if err != nil {
		return nil, err
	}

	return RunRequestInContainer(req, containerId)
}

// ------------------------------------------------

type RunRequest struct {
	ctx context.Context

	execPrefix   string
	lang         string
	code         string
	codeDuration time.Duration

	containerNamePrefix string
	containerVolPrefix  string

	cbAdminPassword string
}

func RunRequestInContainer(req RunRequest, containerId int) (
	[]byte, error) {
	if containerId < 0 {
		return nil, fmt.Errorf("bad containerId: %d", containerId)
	}

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
			containerName,
			"/run-code.sh", req.execPrefix, codePathInst)
	} else {
		cmd = exec.Command("docker", "exec",
			containerName,
			"/run-code.sh", codePathInst)
	}

	log.Printf("INFO: RunRequest, containerId: %d, lang: %s\n",
		containerId, req.lang)

	return ExecCmd(req.ctx, cmd, req.codeDuration)
}

// ------------------------------------------------

// Run a cmd, waiting for it to finish or timeout,
// returning its combined stdout and stderr result.
func ExecCmd(ctx context.Context, cmd *exec.Cmd,
	duration time.Duration) ([]byte, error) {
	var bout, berr []byte

	sout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("cmd.StdoutPipe, err: %v", err)
	}

	serr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("cmd.StderrPipe, err: %v", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("cmd.Start, err: %v", err)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		bout, _ = ioutil.ReadAll(sout)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		berr, _ = ioutil.ReadAll(serr)
	}()

	doneCh := make(chan error, 1)
	go func() {
		wg.Wait()

		doneCh <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		cmd.Process.Kill()
		return nil, fmt.Errorf("ctx.Done, err: %v", ctx.Err())

	case <-time.After(duration):
		cmd.Process.Kill()
		return nil, fmt.Errorf("timeout, duration: %v", duration)

	case err = <-doneCh:
		if err != nil {
			err = fmt.Errorf("ExecCmd, err: %v", err)
		}
	}

	return append(bout, berr...), err
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

// ------------------------------------------------

func AddRBACUser(req RunRequest, containerId int,
	username, password, roles string) error {
	containerName := fmt.Sprintf("%s%d",
		req.containerNamePrefix, containerId)

	cmd := exec.Command("docker", "exec", containerName,
		"/opt/couchbase/bin/couchbase-cli", "user-manage",
		"--cluster", "http://127.0.0.1",
		"--username", "Administrator",
		"--password", req.cbAdminPassword,
		"--set",
		"--rbac-username", username,
		"--rbac-password", password,
		"--auth-domain", "local",
		"--roles", roles)

	// Example out: "SUCCESS: User abcdef set".
	out, err := ExecCmd(req.ctx, cmd, req.codeDuration)
	if err != nil || !strings.HasPrefix(string(out), "SUCCESS:") {
		return fmt.Errorf("AddRBACUser,"+
			" out: %s, err: %v", out, err)
	}

	return nil
}

// ------------------------------------------------

func KillUserProcesses(ctx context.Context,
	containerNamePrefix string, containerId int,
	user string, duration time.Duration) ([]byte, error) {
	containerName := fmt.Sprintf("%s%d",
		containerNamePrefix, containerId)

	cmd := exec.Command("docker", "exec", containerName,
		"pkill", "-u", user)

	return ExecCmd(ctx, cmd, duration)
}
