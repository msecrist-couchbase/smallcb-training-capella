package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func RunRequestSession(session *Session, req RunRequest,
	readyCh chan int, containerWaitDuration time.Duration,
	restartCh chan<- Restart) (out []byte, err error) {
	session, err = SessionAssignContainer(session, req,
		readyCh, containerWaitDuration, restartCh)
	if err != nil {
		return nil, fmt.Errorf("RunRequestSession,"+
			" could not assign container, err: %v", err)
	}

	if session == nil {
		return nil, fmt.Errorf("RunRequestSession, no session")
	}

	out, err = RunRequestInContainer(req, session.ContainerId)

	// Ignore error from the async pkill, as fast running
	// code will have no leftover processes to kill.
	go KillUserProcesses(context.Background(),
		req.containerNamePrefix, session.ContainerId,
		"play", 5*time.Second)

	return out, err
}

// ------------------------------------------------

func SessionAssignContainer(session *Session, req RunRequest,
	readyCh chan int, containerWaitDuration time.Duration,
	restartCh chan<- Restart) (*Session, error) {
	if session == nil {
		return nil, fmt.Errorf("SessionAssignContainer, no session")
	}

	if session.ContainerId >= 0 {
		return session, nil
	}

	containerId, err := WaitForReadyContainer(
		req.ctx, readyCh, containerWaitDuration)
	if err != nil {
		return session, err
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
			if session.ContainerId < 0 {
				session.ContainerId = containerId
				session.RestartCh = restartCh
				session.ReadyCh = readyCh
				session.TouchedAtUnix = time.Now().Unix()

				log.Printf("run_session, SessionAssignContainer,"+
					" sessionId: %s, containerId: %d",
					session.SessionId, session.ContainerId)

				// Session owns the containerId.
				containerId = -1
			}

			rv := *session // Copy.

			return &rv
		})

	return session, nil
}
