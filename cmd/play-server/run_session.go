package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func RunRequestSession(session *Session, req RunRequest,
	readyCh chan int, containerWaitDuration time.Duration,
	restartCh chan<- Restart,
	containers, containersSingleUse int) (out []byte, err error) {
	session, err = SessionAssignContainer(session, req,
		readyCh, containerWaitDuration, restartCh,
		containers, containersSingleUse, "", "", "")
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
	containers, containersSingleUse int,
	init, initKey, defaultBucket string) (*Session, error) {
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

	err = StartInit(req, containerId, init, initKey)
	if err != nil {
		return nil, err
	}

	err = StartCbsh(session, req, containerId, defaultBucket)
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

					log.Printf("run_session, SessionAssignContainer,"+
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

func StartInit(req RunRequest, containerId int, init, initKey string) error {
	if init == "" {
		return nil
	}

	if strings.Index(init, "..") >= 0 {
		log.Printf("ERROR: StartInit, bad init: %s", init);

		return fmt.Errorf("StartInit, bad init");
	}

	data, err := ReadYaml(*staticDir + "/" + init + ".yaml")
	if err != nil {
		return err
	}

	v, ok := CleanupInterfaceValue(data).(map[string]interface{})[initKey]
	if !ok {
		return nil
        }

	m, ok := v.(map[string]interface{})
	if !ok {
		log.Printf("ERROR: StartInit, bad initKey: %s, init: %s, v: %+v", initKey, init, v)

		return fmt.Errorf("StartInit, bad initKey");
	}

	init_req := req // Copy.

	if _, ok := m["execPrefix"]; ok {
		init_req.execPrefix = m["execPrefix"].(string)
	}

	init_req.lang = m["lang"].(string)
	init_req.code = m["code"].(string)
	init_req.codeDuration = *sessionsMaxAge
	init_req.user = "couchbase"

	_, err = RunRequestInContainer(init_req, containerId)
	if err != nil {
		log.Printf("ERROR: StartInit, err: %v", err)
	}

	return err
}

// ------------------------------------------------

func StartCbsh(session *Session, req RunRequest, containerId int, defaultBucket string) error {
	containerName := fmt.Sprintf("%s%d",
		req.containerNamePrefix, containerId)

	// Ex: "vol-instances/vol-0".
	dir := fmt.Sprintf("%s%d",
		req.containerVolPrefix, containerId)

	err := os.MkdirAll(dir+DirCode, 0777)
	if err != nil {
		return err
	}

	// Ex: "vol-instances/vol-0/tmp/cbsh-config".
	cbshConfigHost := dir + "/tmp/cbsh-config"

	// Ex: "/opt/couchbase/var/tmp/cbsh-config".
	cbshConfigInst := DirVar + "/tmp/cbsh-config"


	cbshConfigBytes := []byte(
		"version = 1\n\n" +
			"[clusters.default]\n" +
			"hostnames = [\"127.0.0.1\"]\n" +
			"username = \"" + session.CBUser + "\"\n" +
			"password = \"" + session.CBPswd + "\"\n")

	if defaultBucket != "" && defaultBucket != "NONE" {
		cbshConfigBytes = append(cbshConfigBytes, []byte("default-bucket = \"" + defaultBucket + "\"\n")...)
	}

	// File mode 0777 executable, for scripts like 'code.py'.
	err = ioutil.WriteFile(cbshConfigHost, cbshConfigBytes, 0777)
	if err != nil {
		return err
	}

	cmd := exec.Command("docker", "exec",
		"-detach", "-it", "-u", "cbsh", "-w", "/home/cbsh", containerName,
		"/bin/sh", "-c",
		"mkdir -p /home/cbsh/.cbsh;"+
			" cp "+cbshConfigInst+" /home/cbsh/.cbsh/config;"+
			" while true; do /home/play/npm_packages/bin/gritty --command ./cbsh; sleep 3; done")

	out, err := ExecCmd(req.ctx, cmd, req.codeDuration)
	if err != nil {
		return fmt.Errorf("StartCbsh,"+
			" out: %s, err: %v", out, err)
	}

	return nil
}

// ------------------------------------------------

func RetrieveIP(req RunRequest, containerId int) (string, error) {
	containerName := fmt.Sprintf("%s%d",
		req.containerNamePrefix, containerId)

	cmd := exec.Command("docker", "exec", containerName, "hostname", "-I")

	out, err := ExecCmd(req.ctx, cmd, req.codeDuration)
	if err != nil {
		return "", fmt.Errorf("RetrieveIP,"+
			" out: %s, err: %v", out, err)
	}

	return strings.Trim(string(out), " \n"), nil
}
