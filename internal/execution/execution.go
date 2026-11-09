package execution

import (
	"context"
	"io"
	"sync"
	"time"
)

type execution struct {
	manager   *Manager
	id        string
	command   Command
	started   time.Time
	startedAt time.Time
	ctx       context.Context
	cancel    context.CancelCauseFunc

	mu                     sync.Mutex
	state                  state
	reserved               bool
	event                  terminalEvent
	committed              bool
	status                 int
	dispatchedAt           time.Time
	streaming              *bool
	routingOverheadNS      *int64
	providerLatencyNS      *int64
	timeToFirstTokenNS     *int64
	outputStreamDurationNS *int64
	quotaResetIn           *int64
	firstOutputAt          time.Time
	telemetryEvents        []TelemetryEvent

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
	started := time.Now()
	return &execution{
		manager:           manager,
		id:                id,
		command:           command,
		started:           started,
		startedAt:         started.UTC(),
		ctx:               ctx,
		cancel:            cancel,
		state:             stateAccepted,
		stopReader:        make(chan struct{}),
		routingOverheadNS: command.RoutingOverheadNS,
		telemetryEvents:   initialTelemetryEvents(command),
	}
}

func initialTelemetryEvents(command Command) []TelemetryEvent {
	if command.RoutingSelectedAt.IsZero() {
		return nil
	}

	return []TelemetryEvent{{
		Kind:       EventRoutingSelected,
		ObservedAt: command.RoutingSelectedAt.UTC(),
		ProviderID: command.Target.ProviderID,
		ModelID:    command.Target.ModelID,
		DurationNS: command.RoutingOverheadNS,
	}}
}
