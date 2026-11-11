package execution

import "io"

func (e *execution) attachBody(body io.ReadCloser) {
	e.bodyMu.Lock()
	e.body = body
	closeAsked := e.bodyCloseAsked
	e.bodyMu.Unlock()
	if closeAsked {
		_ = e.closeBody()
	}
}

func (e *execution) currentBody() io.ReadCloser {
	e.bodyMu.Lock()
	defer e.bodyMu.Unlock()
	return e.body
}

func (e *execution) closeBody() error {
	e.bodyMu.Lock()
	e.bodyCloseAsked = true
	if e.body == nil {
		e.bodyMu.Unlock()
		return nil
	}
	if e.bodyCloseStarted {
		done := e.bodyCloseDone
		e.bodyMu.Unlock()
		<-done
		e.bodyMu.Lock()
		err := e.bodyCloseErr
		e.bodyMu.Unlock()
		return err
	}
	e.bodyCloseStarted = true
	e.bodyCloseDone = make(chan struct{})
	done := e.bodyCloseDone
	body := e.body
	e.bodyMu.Unlock()

	err := body.Close()
	e.bodyMu.Lock()
	e.bodyCloseErr = err
	close(done)
	e.bodyMu.Unlock()
	return err
}
