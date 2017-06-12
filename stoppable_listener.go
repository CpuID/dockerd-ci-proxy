package main

import (
	"errors"
	"net"
	"time"
)

// Credit: http://www.hydrogen18.com/blog/stop-listening-http-server-go.html

// A stoppable UNIX listener, using a channel for shutdown handling.
type stoppableUnixListener struct {
	*net.UnixListener          // Wrapped UNIX listener
	stop              chan int // Channel used only to indicate listener should shutdown
}

func newStoppableUnixListener(listener net.Listener) (*stoppableUnixListener, error) {
	new_listener, ok := listener.(*net.UnixListener)
	if !ok {
		return nil, errors.New("Cannot wrap UNIX listener")
	}

	retval := &stoppableUnixListener{}
	retval.UnixListener = new_listener
	retval.stop = make(chan int)

	return retval, nil
}

func (sl *stoppableUnixListener) Accept() (net.Conn, error) {
	for {
		// Wait up to 500ms for a new connection
		sl.SetDeadline(time.Now().Add(500 * time.Millisecond))

		new_conn, err := sl.UnixListener.Accept()

		// Check for the channel being closed
		select {
		case <-sl.stop:
			return nil, errors.New("Listener Stopped")
		default:
			// If the channel is still open, continue as normal
		}

		if err != nil {
			net_err, ok := err.(net.Error)

			// If this is a timeout, then continue to wait for new connections
			if ok && net_err.Timeout() && net_err.Temporary() {
				continue
			}
		}

		return new_conn, err
	}
}

func (sl *stoppableUnixListener) Stop() {
	close(sl.stop)
}
