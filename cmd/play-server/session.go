package main

import "fmt"
import "strings"
import "sync"

import "github.com/google/uuid"

type Sessions struct {
	m sync.Mutex // Protects the fields that follow.

	mapBySessionId     map[string]*Session
	mapByFullNameEmail map[string]string // Value is sessionId.
}

type Session struct {
	SessionIdent

	ContainerId int
	RestartCh   chan<- Restart
}

type SessionIdent struct {
	SessionId string

	FullName string
	Email    string

	CBUser string
	CBPswd string
}

// ------------------------------------------------

var sessions = Sessions{
	mapBySessionId:     map[string]*Session{},
	mapByFullNameEmail: map[string]string{},
}

// ------------------------------------------------

func (sessions *Sessions) SessionGet(sessionId string) *Session {
	StatsNumInc("tot.sessions.SessionGet")

	rv := sessions.SessionAccess(sessionId, func(session *Session) *Session {
		rv := *session // Returns a copy.

		return &rv
	})

	if rv == nil {
		StatsNumInc("tot.sessions.SessionGet.nil")
	} else {
		StatsNumInc("tot.sessions.SessionGet.ok")
	}

	return rv
}

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

func (sessions *Sessions) SessionExit(sessionId string) error {
	StatsNumInc("tot.sessions.SessionExit")

	sessions.m.Lock()
	defer sessions.m.Unlock()

	session, exists := sessions.mapBySessionId[sessionId]
	if exists && session != nil {
		delete(sessions.mapBySessionId, sessionId)
		delete(sessions.mapByFullNameEmail,
			FullNameEmail(session.FullName, session.Email))
	}

	return nil
}

func (s *Sessions) SessionCreate(fullName, email string) (sessionId string, err error) {
	StatsNumInc("tot.sessions.SessionCreate")

	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		StatsNumInc("tot.sessions.SessionCreate.err.ErrNeedFullName")

		return "", fmt.Errorf("ErrNeedFullName")
	}

	email = strings.TrimSpace(email)
	if email == "" {
		StatsNumInc("tot.sessions.SessionCreate.err.ErrNeedEmail")

		return "", fmt.Errorf("ErrNeedEmail")
	}

	fullNameEmail := FullNameEmail(fullName, email)

	sessions.m.Lock()
	defer sessions.m.Unlock()

	sessionId, exists := sessions.mapByFullNameEmail[fullNameEmail]
	if exists || sessionId != "" {
		StatsNumInc("tot.sessions.SessionCreate.err.ErrFullNameEmailUsed")

		return "", fmt.Errorf("ErrFullNameEmailUsed")
	}

	session, exists := sessions.mapBySessionId[sessionId]
	if exists || session != nil {
		StatsNumInc("tot.sessions.SessionCreate.err.ErrSessionIdUsed")

		return "", fmt.Errorf("ErrSessionIdUsed")
	}

	sessionUUID, err := uuid.NewRandom()
	if err != nil {
		StatsNumInc("tot.sessions.SessionCreate.err.NewRandom")

		return "", err
	}

	sessionId = strings.ReplaceAll(sessionUUID.String(), "-", "")

	session = &Session{
		SessionIdent: SessionIdent{
			SessionId: sessionId,
			FullName:  fullName,
			Email:     email,
			CBUser:    sessionId[:16],
			CBPswd:    sessionId[16:],
		},
		ContainerId: -1,
	}

	sessions.mapBySessionId[sessionId] = session
	sessions.mapByFullNameEmail[fullNameEmail] = sessionId

	StatsNumInc("tot.sessions.SessionCreate.ok")

	return sessionId, nil
}

// ------------------------------------------------

func FullNameEmail(fullName, email string) string {
	return fullName + "-" + email
}
