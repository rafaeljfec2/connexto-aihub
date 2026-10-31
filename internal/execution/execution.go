package execution

import (
	"context"
	"io"
	"sync"
	"time"
)

type execution struct {
	manager *Manager
	id      string
	command Command
	started time.Time
	ctx     context.Context
	cancel  context.CancelCauseFunc

	mu        sync.Mutex
	state     state
	reserved  bool
	event     terminalEvent
	committed bool
	status    int

	stopReader chan struct{}
	stopOnce   sync.Once
	readerDone chan struct{}

	bodyMu           sync.Mutex
	body             io.ReadCloser
	bodyCloseAsked   bool
	bodyCloseStarted bool
	bodyCloseDone    chan struct{}
	bodyCloseErr     error
}

type readResult struct {
	data []byte
	err  error
}

func newExecution(manager *Manager, id string, command Command, parent context.Context) *execution {
	ctx, cancel := context.WithCancelCause(parent)
	command.Payload = append([]byte(nil), command.Payload...)
	return &execution{
		manager:    manager,
		id:         id,
		command:    command,
		started:    time.Now(),
		ctx:        ctx,
		cancel:     cancel,
		state:      stateAccepted,
		stopReader: make(chan struct{}),
	}
}
