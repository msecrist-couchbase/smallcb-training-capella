package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

func RunRequestSessionCbsh(session *Session, req RunRequest,
	readyCh chan int, containerWaitDuration time.Duration,
	restartCh chan<- Restart,
	containers, containersSingleUse int) (out []byte, err error) {
	session, err = SessionAssignContainer(session, req,
		readyCh, containerWaitDuration, restartCh,
		containers, containersSingleUse, "", "", "")
	if err != nil {
		return nil, fmt.Errorf("RunRequestSessionCbsh,"+
			" could not assign container, err: %v", err)
	}

	if session == nil {
		return nil, fmt.Errorf("RunRequestSessionCbsh, no session")
	}

	// Ignore error from the async pkill, as fast running
	// code will have no leftover processes to kill.
	go KillUserProcesses(context.Background(),
		req.containerNamePrefix, session.ContainerId,
		"play", 5*time.Second)

	return out, err
}

// ------------------------------------------------

func SessionAssignContainerCbsh(session *Session, req RunRequest,
	readyCh chan int, containerWaitDuration time.Duration,
	restartCh chan<- Restart,
	containers, containersSingleUse int,
	init, initKey, defaultBucket string, Target target) (*Session, error) {
	if session == nil {
		return nil, fmt.Errorf("SessionAssignContainerCbsh, no session")
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

		return session, fmt.Errorf("SessionAssignContainerCbsh," +
			" no container available for your Cbsh session")
	}

	defer func() {
		if containerId >= 0 {
			restartCh <- Restart{
				ContainerId: containerId,
				ReadyCh:     readyCh,
			}
		}
	}()

	if &Target != nil && Target.DBurl != "" {
		Target.DBurl = strings.ReplaceAll(Target.DBurl, "couchbase://", "")
		Target.DBurl = strings.ReplaceAll(Target.DBurl, "couchbases://", "")
		if strings.Contains(Target.DBurl, "?") {
			Target.DBurl = strings.Split(Target.DBurl, "?")[0]
		}
		session.CBHost = Target.DBurl
		session.CBUser = Target.DBuser
		session.CBPswd = Target.DBpwd
	}
	err = StartCbsh(session, req, containerId, defaultBucket)
	if err != nil {
		return nil, err
	}

	err = StartToolsTerminal(session, req, containerId, defaultBucket)
	if err != nil {
		return nil, err
	}
	containerIP, err := RetrieveIP(req, containerId)
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
					session.ContainerIP = containerIP
					session.ContainerPortBase =
						*listenPortBase + (containerId * *listenPortSpan)

					session.RestartCh = restartCh
					session.ReadyCh = readyCh
					session.TouchedAtUnix = time.Now().Unix()

					log.Printf("run_session, SessionAssignContainerCbsh,"+
						" sessionId: %s, containerId: %d, containerIP: %s",
						session.SessionId,
						session.ContainerId,
						session.ContainerIP)

					// Session owns the containerId.
					containerId = -1
				}
			}

			rv := *session // Copy.

			return &rv
		})

	return session, err
}

// ------------------------------------------------
