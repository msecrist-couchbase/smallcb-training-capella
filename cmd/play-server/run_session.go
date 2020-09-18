package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func RunRequestSession(session *Session, req RunRequest,
	readyCh chan int, containerWaitDuration time.Duration,
	restartCh chan<- Restart,
	containers, containersSingleUse int) (out []byte, err error) {
	session, err = SessionAssignContainer(session, req,
		readyCh, containerWaitDuration, restartCh,
		containers, containersSingleUse)
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
	restartCh chan<- Restart,
	containers, containersSingleUse int) (*Session, error) {
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

	// Race-y check here to see if we're already below containersSingleUse,
	// with another real check later in the protected SessionsAccess() call,
	// where this early check is cheaper as a full container instance
	// restart isn't needed at this point.
	_, sessionsCountWithContainer := sessions.Count()
	if containers-int(sessionsCountWithContainer)-1 < containersSingleUse {
		readyCh <- containerId

		containerId = -1

		return session, fmt.Errorf("SessionAssignContainer," +
			" no container available for your session")
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
				_, sessionsCountWithContainer := sessions.CountLOCKED()
				if containers-int(sessionsCountWithContainer)-1 <
					containersSingleUse {
					err = fmt.Errorf("SessionAssignContainer," +
						" no container left for your session")
				} else {
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
			}

			rv := *session // Copy.

			return &rv
		})

	return session, err
}
