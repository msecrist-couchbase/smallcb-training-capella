package main

import (
	"fmt"
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
			if containerId >= 0 {
				restartCh <- Restart{
					ContainerId: containerId,
					ReadyCh:     readyCh,
				}
			}
		}()

		err = AddRBACUser(req, containerId,
			session.CBUser, session.CBPswd, "admin")
		if err != nil {
			return nil, err
		}

		session = sessions.SessionAccess(session.SessionId,
			func(session *Session) *Session {
				session.ContainerId = containerId
				session.RestartCh = restartCh
				session.ReadyCh = readyCh
				session.TouchedAt = time.Now()

				// Session owns the containerId.
				containerId = -1

				rv := *session // Copy.

				return &rv
			})
	}

	if session == nil {
		return nil, fmt.Errorf("RunRequestSession, no session")
	}

	return RunRequestInContainer(req, session.ContainerId)
}
