package main

import (
	"context"
	"fmt"
	"time"
)

func RunLangCodeSession(ctx context.Context, session *Session, user,
	execPrefix, lang, code string, codeDuration time.Duration,
	readyCh chan int,
	containerWaitDuration time.Duration,
	containerNamePrefix,
	containerVolPrefix string,
	restartCh chan<- Restart) ([]byte, error) {
	if session != nil && session.ContainerId < 0 {
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

		session = sessions.SessionAccess(session.SessionId,
			func(session *Session) *Session {
				session.ContainerId = containerId
				session.RestartCh = restartCh

				rv := *session // Copy.

				containerId = -1 // Session now owns the containerId.

				return &rv
			})
	}

	if session == nil {
		return nil, fmt.Errorf("RunLangCodeSession, no session")
	}

	result, err := RunLangCodeContainer(ctx, user,
		execPrefix, lang, code, codeDuration,
		session.ContainerId, containerNamePrefix, containerVolPrefix)

	return result, err
}
