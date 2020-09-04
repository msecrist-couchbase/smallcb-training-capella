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
	// TODO.

	// Atomically grab a containerId token, blocking
	// until a container instance is ready or we timeout.
	var containerId int

	select {
	case containerId = <-readyCh:
		defer func() {
			// If we didn't use the container and mess it up,
			// we can immediately put the containerId token
			// back into the readyCh for the next request.
			if containerId >= 0 {
				readyCh <- containerId
			}
		}()

	case <-time.After(containerWaitDuration):
		return nil, fmt.Errorf("timeout waiting for container instance,"+
			" duration: %v", containerWaitDuration)

	case <-ctx.Done():
		// Client canceled/timed-out while we were waiting.
		return nil, ctx.Err()
	}

	result, err := RunLangCodeContainer(ctx, user,
		execPrefix, lang, code, codeDuration,
		containerId, containerNamePrefix, containerVolPrefix)

	go func(containerId int) {
		restartCh <- Restart{
			ContainerId: containerId,
			DoneCh:      readyCh,
		}
	}(containerId)

	// The restartCh now owns the containerId token.
	containerId = -1

	return result, err
}
