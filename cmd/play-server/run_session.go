package main

import (
	"context"
	"time"
)

func RunLangCodeSession(ctx context.Context, session *Session, user,
	execPrefix, lang, code string, codeDuration time.Duration,
	readyCh chan int,
	containerWaitDuration time.Duration,
	containerNamePrefix,
	containerVolPrefix string,
	restartCh chan<- Restart) ([]byte, error) {
	// TODO.

	result, err := RunLangCode(ctx,
		ExecUser, ExecPrefixes[lang],
		lang, code, codeDuration, readyCh,
		containerWaitDuration,
		containerNamePrefix,
		containerVolPrefix,
		restartCh)

	return result, err
}
