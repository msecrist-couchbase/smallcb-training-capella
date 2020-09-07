package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Sessions struct {
	m sync.Mutex // Protects the fields that follow.

	mapBySessionId     map[string]*Session
	mapByFullNameEmail map[string]string // Value is sessionId.
}

type Session struct {
	SessionInfo

	// The following fields are ephemeral / runtime-only.

	ContainerId int

	RestartCh chan<- Restart
	ReadyCh   chan int

	Cookies []string
}

// SessionInfo fields are intended to be persistable.
type SessionInfo struct {
	SessionId string

	FullName string
	Email    string

	CBUser string
	CBPswd string

	TouchedAt int64
}

// ------------------------------------------------

var sessions = Sessions{
	mapBySessionId:     map[string]*Session{},
	mapByFullNameEmail: map[string]string{},
}

// ------------------------------------------------

func (sessions *Sessions) Count() (count, countWithContainer uint64) {
	sessions.m.Lock()

	count = uint64(len(sessions.mapBySessionId))

	for _, session := range sessions.mapBySessionId {
		if session.ContainerId >= 0 {
			countWithContainer += 1
		}
	}

	sessions.m.Unlock()

	return count, countWithContainer
}

// ------------------------------------------------

func (sessions *Sessions) SessionGet(sessionId string) *Session {
	StatsNumInc("sessions.SessionGet")

	rv := sessions.SessionAccess(sessionId,
		func(session *Session) *Session {
			session.TouchedAt = time.Now().Unix()

			rv := *session // Returns a copy.

			return &rv
		})

	if rv == nil {
		StatsNumInc("sessions.SessionGet.nil")
	} else {
		StatsNumInc("sessions.SessionGet.ok")
	}

	return rv
}

// ------------------------------------------------

func (sessions *Sessions) SessionAccess(sessionId string,
	cb func(*Session) *Session) *Session {
	sessions.m.Lock()
	defer sessions.m.Unlock()

	session, exists := sessions.mapBySessionId[sessionId]
	if !exists || session == nil {
		return nil
	}

	return cb(session)
}

// ------------------------------------------------

func (sessions *Sessions) SessionExit(sessionId string) error {
	StatsNumInc("sessions.SessionExit")

	sessions.m.Lock()
	defer sessions.m.Unlock()

	session, exists := sessions.mapBySessionId[sessionId]
	if exists && session != nil {
		delete(sessions.mapBySessionId, sessionId)

		delete(sessions.mapByFullNameEmail,
			FullNameEmail(session.FullName, session.Email))

		go func() { // Async to not hold the sessions lock.
			session.ReleaseContainer()

			CookiesRemove(session.Cookies)
		}()

		StatsNumInc("sessions.SessionExit.ok")
	} else {
		StatsNumInc("sessions.SessionExit.none")
	}

	return nil
}

// ------------------------------------------------

func (s *Sessions) SessionCreate(fullName, email string) (
	session *Session, err error) {
	StatsNumInc("sessions.SessionCreate")

	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		StatsNumInc("sessions.SessionCreate.err.ErrNeedFullName")

		return nil, fmt.Errorf("ErrNeedFullName")
	}

	email = strings.TrimSpace(email)
	if email == "" {
		StatsNumInc("sessions.SessionCreate.err.ErrNeedEmail")

		return nil, fmt.Errorf("ErrNeedEmail")
	}

	fullNameEmail := FullNameEmail(fullName, email)

	sessions.m.Lock()
	defer sessions.m.Unlock()

	sessionId, exists := sessions.mapByFullNameEmail[fullNameEmail]
	if exists || sessionId != "" {
		StatsNumInc("sessions.SessionCreate.err.ErrFullNameEmailUsed")

		return nil, fmt.Errorf("ErrFullNameEmailUsed")
	}

	session, exists = sessions.mapBySessionId[sessionId]
	if exists || session != nil {
		StatsNumInc("sessions.SessionCreate.err.ErrSessionIdUsed")

		return nil, fmt.Errorf("ErrSessionIdUsed")
	}

	sessionUUID, err := uuid.NewRandom()
	if err != nil {
		StatsNumInc("sessions.SessionCreate.err.NewRandom")

		return nil, err
	}

	sessionId = strings.ReplaceAll(sessionUUID.String(), "-", "")

	session = &Session{
		SessionInfo: SessionInfo{
			SessionId: sessionId,
			FullName:  fullName,
			Email:     email,
			CBUser:    sessionId[:16],
			CBPswd:    sessionId[16:],
			TouchedAt: time.Now().Unix(),
		},
		ContainerId: -1,
	}

	sessions.mapBySessionId[sessionId] = session
	sessions.mapByFullNameEmail[fullNameEmail] = sessionId

	StatsNumInc("sessions.SessionCreate.ok")

	rv := *session // Copy

	return &rv, nil
}

// ------------------------------------------------

func FullNameEmail(fullName, email string) string {
	return fullName + "-" + email
}

// ------------------------------------------------

// Release a given number of containers held by
// sessions, and asynchronously restart those
// container instances -- use -1 to release all the
// container instances.
func (sessions *Sessions) ReleaseContainers(n int) int {
	var ss []Session

	sessions.m.Lock()

	for _, session := range sessions.mapBySessionId {
		if n == 0 {
			break
		}

		if session.ContainerId >= 0 {
			ss = append(ss, *session) // Copy.

			session.ContainerId = -1
			session.RestartCh = nil
			session.ReadyCh = nil

			n -= 1
		}
	}

	sessions.m.Unlock()

	go func() {
		for _, s := range ss {
			s.ReleaseContainer()
		}
	}()

	return len(ss)
}

// ------------------------------------------------

func (s *Session) ReleaseContainer() {
	if s.ContainerId >= 0 {
		StatsNumInc("session.ReleaseContainer")

		s.RestartCh <- Restart{
			ContainerId: s.ContainerId,
			ReadyCh:     s.ReadyCh,
		}

		StatsNumInc("session.ReleaseContainer.sent")
	}

	s.ContainerId = -1
	s.RestartCh = nil
	s.ReadyCh = nil
}

// ------------------------------------------------

func SessionsChecker(sleepDur, maxAgeDur time.Duration) {
	var sessionIds []string

	for {
		sessionIds = sessionIds[:0]

		time.Sleep(sleepDur)

		cutOffTime := time.Now().Add(-maxAgeDur)

		sessions.m.Lock()

		for _, session := range sessions.mapBySessionId {
			if time.Unix(session.TouchedAt, 0).Before(cutOffTime) {
				sessionIds = append(sessionIds, session.SessionId)
			}
		}

		sessions.m.Unlock()

		for _, sessionId := range sessionIds {
			sessions.SessionExit(sessionId)
		}
	}
}
