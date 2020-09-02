package main

import "fmt"
import "sync"

type Sessions struct {
	m sync.Mutex

	mapBySessionId     map[string]*Session
	mapByFullNameEmail map[string]string // Value is sessionId.
}

type Session struct {
	SessionIdent

	State       string
	ContainerId int
}

type SessionIdent struct {
	SessionId string
	FullName  string
	Email     string
}

// ------------------------------------------------

var sessions = Sessions{
	mapBySessionId:     map[string]*Session{},
	mapByFullNameEmail: map[string]string{},
}

// ------------------------------------------------

func (sessions *Sessions) SessionExit(sessionId string) error {
	sessions.m.Lock()
	defer sessions.m.Unlock()

	session, exists := sessions.mapBySessionId[sessionId]
	if exists && session != nil {
		delete(sessions.mapBySessionId, sessionId)
		delete(sessions.mapByFullNameEmail, FullNameEmail(session.FullName, session.Email))
	}

	return nil
}

func (s *Sessions) SessionCreate(fullName, email string) (sessionId string, err error) {
	fullNameEmail := FullNameEmail(fullName, email)

	sessions.m.Lock()
	defer sessions.m.Unlock()

	sessionId, exists := sessions.mapByFullNameEmail[fullNameEmail]
	if exists || sessionId != "" {
		return "", fmt.Errorf("ErrFullNameEmailUsed")
	}

	session, exists := sessions.mapBySessionId[sessionId]
	if exists || session != nil {
		return "", fmt.Errorf("ErrSessionIdExists")
	}

	// TODO: Better sessionId generator / UUID.
	sessionId = fmt.Sprintf("%d-%s", len(sessions.mapBySessionId), fullNameEmail)

	session = &Session{
		SessionIdent: SessionIdent{
			SessionId: sessionId,
			FullName:  fullName,
			Email:     email,
		},
	}

	sessions.mapBySessionId[sessionId] = session
	sessions.mapByFullNameEmail[fullNameEmail] = sessionId

	return sessionId, nil
}

// ------------------------------------------------

func SessionExit(sessionId string) error {
	return sessions.SessionExit(sessionId)
}

func SessionCreate(fullName, email string) (sessionId string, err error) {
	if fullName == "test-bad" {
		return "", fmt.Errorf("ErrTestBad")
	}

	return sessions.SessionCreate(fullName, email)
}

func FullNameEmail(fullName, email string) string {
	return fullName + "-" + email
}
