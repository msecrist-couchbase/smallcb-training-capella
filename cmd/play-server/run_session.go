package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func RunRequestSession(session *Session, req RunRequest,
	readyCh chan int, containerWaitDuration time.Duration,
	restartCh chan<- Restart) ([]byte, error) {
	if session != nil && session.ContainerId < 0 {
		containerId, err := WaitForReadyContainer(
			req.ctx, readyCh, containerWaitDuration)
		if err != nil {
			return nil, err
		}

		defer func() {
			// We restart the container instance if we
			// didn't end up using it -- just in case if
			// we got through the rbac user creation step.
			if containerId >= 0 {
				restartCh <- Restart{
					ContainerId: containerId,
					ReadyCh:     readyCh,
				}
			}
		}()

		containerName := fmt.Sprintf("%s%d",
			req.containerNamePrefix, containerId)

		cmd := exec.Command("docker", "exec", "-u", req.execUser,
			containerName,
			"/opt/couchbase/bin/couchbase-cli", "user-manage",
			"--cluster", "http://127.0.0.1",
			"--username", "Administrator",
			"--password", "password",
			"--set",
			"--rbac-username", session.CBUser,
			"--rbac-password", session.CBPswd,
			"--auth-domain", "local",
			"--roles", "admin")

		// Example out: "SUCCESS: User a637c2348544 set".
		out, err := ExecCmd(req.ctx, cmd, req.codeDuration)
		if err != nil || !strings.HasPrefix(string(out), "SUCCESS:") {
			return nil, fmt.Errorf("RunRequestSession, user-manage,"+
				" out: %s, err: %v", out, err)
		}

		session = sessions.SessionAccess(session.SessionId,
			func(session *Session) *Session {
				session.ContainerId = containerId
				session.RestartCh = restartCh
				session.ReadyCh = readyCh
				session.TouchedAt = time.Now()

				rv := *session // Copy.

				// Session owns the containerId.
				containerId = -1

				return &rv
			})
	}

	if session == nil {
		return nil, fmt.Errorf("RunRequestSession, no session")
	}

	return RunRequestInContainer(req, session.ContainerId)
}
