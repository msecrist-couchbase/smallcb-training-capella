package main

import (
	"fmt"
	"time"
)

func RunRequestSession(session *Session, req RunRequest,
	readyCh chan int,
	containerWaitDuration time.Duration,
	restartCh chan<- Restart) ([]byte, error) {
	if session != nil && session.ContainerId < 0 {
		containerId, err := WaitForReadyContainer(
			req.ctx, readyCh, containerWaitDuration)
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

		session = sessions.SessionAccess(session.SessionId,
			func(session *Session) *Session {
				session.ContainerId = containerId
				session.RestartCh = restartCh
				session.TouchedAt = time.Now()

				rv := *session // Copy.

				containerId = -1 // Session now owns the containerId.

				return &rv
			})
	}

	if session == nil {
		return nil, fmt.Errorf("RunRequestSession, no session")
	}

	return RunRequestInContainer(req, session.ContainerId)
}
