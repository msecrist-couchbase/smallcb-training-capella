package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Sessions struct {
	m sync.Mutex // Protects the fields that follow.

	mapBySessionId map[string]*Session
	mapByNameEmail map[string]string // Value is sessionId.
}

type Session struct {
	SessionInfo

	// The following fields are ephemeral / runtime-only.

	ContainerId int

	ContainerIP string

	RestartCh chan<- Restart
	ReadyCh   chan int

	Cookies []string
}

// SessionInfo fields are intended to be persistable.
type SessionInfo struct {
	SessionId     string
	SessionIdMain string // Can be "".

	Name  string
	Email string

	CBUser string
	CBPswd string

	CreatedAt     string
	CreatedAtUnix int64
	TouchedAtUnix int64
}

// For multi-session scenarios, sessions can be grouped
// together, where the first session in a group becomes
// the main session that has a real name and email,
// where a group can look like...
//
//      SessionId SessionIdMain Name     Email
// main "123"     ""            "a"      "a@email.com"
// 2nd  "234"     "123"         "~123-1" "~"
// 3rd  "456"     "123"         "~123-2" "~"

// ------------------------------------------------

var sessions = Sessions{
	mapBySessionId: map[string]*Session{},
	mapByNameEmail: map[string]string{},
}

// ------------------------------------------------

func (sessions *Sessions) Count() (count, countWithContainer uint64) {
	sessions.m.Lock()

	count, countWithContainer = sessions.CountLOCKED()

	sessions.m.Unlock()

	return count, countWithContainer
}

func (sessions *Sessions) CountLOCKED() (count, countWithContainer uint64) {
	count = uint64(len(sessions.mapBySessionId))

	for _, session := range sessions.mapBySessionId {
		if session.ContainerId >= 0 {
			countWithContainer += 1
		}
	}

	return count, countWithContainer
}

// ------------------------------------------------

func (sessions *Sessions) SessionGet(sessionId string) *Session {
	StatsNumInc("sessions.SessionGet")

	rv := sessions.SessionAccess(sessionId,
		func(session *Session) *Session {
			session.TouchedAtUnix = time.Now().Unix()

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
		sessions.SessionExitLOCKED(session)

		for _, childSession := range sessions.mapBySessionId {
			if childSession.SessionIdMain == session.SessionId {
				sessions.SessionExitLOCKED(childSession)
			}
		}

		StatsNumInc("sessions.SessionExit.ok")
	} else {
		StatsNumInc("sessions.SessionExit.nil")
	}

	return nil
}

// ------------------------------------------------

func (sessions *Sessions) SessionExitLOCKED(session *Session) {
	delete(sessions.mapBySessionId, session.SessionId)

	delete(sessions.mapByNameEmail,
		NameEmail(session.Name, session.Email))

	go func() { // Async to not hold the sessions lock.
		if session.SessionIdMain == "" {
			j, err := json.Marshal(session.SessionInfo)
			if err == nil {
				log.Printf("session, SessionExit, sessionInfo: %+v", string(j))
			}
		}

		session.ReleaseContainer()

		CookiesRemove(session.Cookies)
	}()
}

// ------------------------------------------------

func (s *Sessions) SessionCreate(sessionIdMain, name, email string) (
	session *Session, err error) {
	StatsNumInc("sessions.SessionCreate")

	name = strings.TrimSpace(name)
	if name == "" {
		StatsNumInc("sessions.SessionCreate.err",
			"sessions.SessionCreate.err.ErrNeedName")

		return nil, fmt.Errorf("ErrNeedName")
	}

	email = strings.TrimSpace(email)
	if email == "" {
		StatsNumInc("sessions.SessionCreate.err",
			"sessions.SessionCreate.err.ErrNeedEmail")

		return nil, fmt.Errorf("ErrNeedEmail")
	}

	nameEmail := NameEmail(name, email)

	sessions.m.Lock()
	defer sessions.m.Unlock()

	sessionId, exists := sessions.mapByNameEmail[nameEmail]
	if exists || sessionId != "" {
		StatsNumInc("sessions.SessionCreate.err",
			"sessions.SessionCreate.err.ErrNameEmailUsed")

		return nil, fmt.Errorf("ErrNameEmailUsed")
	}

	session, exists = sessions.mapBySessionId[sessionId]
	if exists || session != nil {
		StatsNumInc("sessions.SessionCreate.err",
			"sessions.SessionCreate.err.ErrSessionIdUsed")

		return nil, fmt.Errorf("ErrSessionIdUsed")
	}

	sessionUUID, err := uuid.NewRandom()
	if err != nil {
		StatsNumInc("sessions.SessionCreate.err",
			"sessions.SessionCreate.err.NewRandom")

		return nil, err
	}

	sessionId = strings.ReplaceAll(sessionUUID.String(), "-", "")

	now := time.Now()

	session = &Session{
		SessionInfo: SessionInfo{
			SessionId:     sessionId,
			SessionIdMain: sessionIdMain,
			Name:          name,
			Email:         email,
			CBUser:        sessionId[:16],
			CBPswd:        sessionId[16:],

			CreatedAt:     now.Format("2006-01-02T15:04:05.000-07:00"),
			CreatedAtUnix: now.Unix(),
			TouchedAtUnix: now.Unix(),
		},
		ContainerId: -1,
	}

	sessions.mapBySessionId[sessionId] = session
	sessions.mapByNameEmail[nameEmail] = sessionId

	StatsNumInc("sessions.SessionCreate.ok")

	if session.SessionIdMain == "" {
		j, err := json.Marshal(session.SessionInfo)
		if err == nil {
			log.Printf("session, SessionCreate, sessionInfo: %+v", string(j))
		}
	}

	rv := *session // Copy

	return &rv, nil
}

// ------------------------------------------------

func NameEmail(name, email string) string {
	return name + "-" + email
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

func SessionsChecker(sleepDur, maxAge, maxIdle time.Duration) {
	var sessionIds []string

	for {
		sessionIds = sessionIds[:0]

		time.Sleep(sleepDur)

		now := time.Now()

		limitCreatedAt := now.Add(-maxAge)
		limitTouchedAt := now.Add(-maxIdle)

		sessions.m.Lock()

		for _, session := range sessions.mapBySessionId {
			if time.Unix(session.CreatedAtUnix, 0).Before(limitCreatedAt) ||
				time.Unix(session.TouchedAtUnix, 0).Before(limitTouchedAt) {
				sessionIds = append(sessionIds, session.SessionId)
			}

		}

		sessions.m.Unlock()

		for _, sessionId := range sessionIds {
			sessions.SessionExit(sessionId)
		}
	}
}
