// Package wsrpc provides a ReadWriteCloser which operates on a WebSocket
// connection.
package wsrpc

import (
	"io"
	"sync"

	"github.com/gorilla/websocket"
)

// ReadWriteCloser is a rwc based on WebSockets
type ReadWriteCloser struct {
	mu sync.Mutex
	ws *websocket.Conn
	r  io.Reader
	w  io.WriteCloser
}

// NewReadWriteCloser creates a new rwc from a WebSocket connection
func NewReadWriteCloser(ws *websocket.Conn) ReadWriteCloser {
	return ReadWriteCloser{ws: ws}
}

// Read reads from the WebSocket into p
func (rwc *ReadWriteCloser) Read(p []byte) (n int, err error) {
	var r io.Reader
	var ws *websocket.Conn

	rwc.mu.Lock()
	ws = rwc.ws
	r = rwc.r
	rwc.mu.Unlock()

	if ws == nil {
		return 0, io.ErrClosedPipe
	}

	if r == nil {
		_, r, err = ws.NextReader()
		if err != nil {
			return 0, err
		}
		rwc.mu.Lock()
		if rwc.ws == nil {
			rwc.mu.Unlock()
			return 0, io.ErrClosedPipe
		}
		rwc.r = r
		rwc.mu.Unlock()
	}

	for n = 0; n < len(p); {
		var m int
		m, err = r.Read(p[n:])
		n += m
		if err == io.EOF {
			// done
			rwc.mu.Lock()
			if rwc.r == r {
				rwc.r = nil
			}
			rwc.mu.Unlock()
			break
		}
		if err != nil {
			break
		}
	}

	return
}

// Write writes the provided bytes to the WebSocket
func (rwc *ReadWriteCloser) Write(p []byte) (n int, err error) {
	var w io.WriteCloser
	var ws *websocket.Conn

	rwc.mu.Lock()
	ws = rwc.ws
	w = rwc.w
	rwc.mu.Unlock()

	if ws == nil {
		return 0, io.ErrClosedPipe
	}

	if w == nil {
		w, err = ws.NextWriter(websocket.TextMessage)
		if err != nil {
			return 0, err
		}
		rwc.mu.Lock()
		if rwc.ws == nil {
			rwc.mu.Unlock()
			_ = w.Close()
			return 0, io.ErrClosedPipe
		}
		rwc.w = w
		rwc.mu.Unlock()
	}

	for n = 0; n < len(p); {
		m, err := w.Write(p[n:])
		n += m
		if err != nil {
			break
		}
	}
	if err != nil || n == len(p) {
		closeErr := w.Close()
		rwc.mu.Lock()
		if rwc.w == w {
			rwc.w = nil
		}
		rwc.mu.Unlock()
		if err == nil {
			err = closeErr
		}
	}

	return
}

// Close the rwc and the underlying WebSocket connection
func (rwc *ReadWriteCloser) Close() error {
	var err error
	var w io.WriteCloser
	var ws *websocket.Conn

	rwc.mu.Lock()
	w = rwc.w
	rwc.w = nil
	rwc.r = nil
	ws = rwc.ws
	rwc.ws = nil
	rwc.mu.Unlock()

	if w != nil {
		if err = w.Close(); err != nil {
			return err
		}
	}
	if ws != nil {
		return ws.Close()
	}
	return nil
}
