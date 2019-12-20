package tunnelmgr

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"

	"github.com/tigera/voltron/pkg/tunnel"

	"github.com/tigera/voltron/pkg/state"
)

// ErrManagerClosed is returned when a closed manager is used
var ErrManagerClosed = fmt.Errorf("manager closed")

// ErrTunnelNotReady is returned when a closed manager is used
var ErrTunnelNotReady = fmt.Errorf("tunnel not ready")

// ErrTunnelSet is returned when the tunnel has already been set and you try to set it again with one of the Run functions
var ErrTunnelSet = fmt.Errorf("tunnel already set")

// Manager is an interface used to manage access to tunnel(s). It synchronises access to the tunnel(s), and abstracts
// out logic necessary to interact with the tunnel(s). The main motivation for this was that both sides of the
// tunnel need to open and accept connections on a single tunnel, so instead of duplicating that logic on both the client
// and server side of the tunnel, it is abstracted out into a single component that both sides can use
type Manager interface {
	RunWithTunnel(t *tunnel.Tunnel) error
	RunWithDialer(t tunnel.Dialer) error
	Open() (net.Conn, error)
	OpenTLS(*tls.Config) (net.Conn, error)
	Listener() (net.Listener, error)
	ListenForErrors() chan error
	CloseTunnel() error
	Close() error
}

type manager struct {
	setTunnel state.SendToStateChan
	setDialer state.SendToStateChan

	openConnection   state.SendToStateChan
	addListener      state.SendToStateChan
	addErrorListener state.SendToStateChan

	closeTunnel state.SendToStateChan
	// this is used to notify the listener that the manager is closed
	close chan bool

	closeOnce sync.Once
}

// NewManager returns an instance of the Manager interface. Use one of the Run functions to start sending / accepting
// connections over a tunnel
func NewManager() Manager {
	m := &manager{}
	m.setTunnel = make(state.SendToStateChan)
	m.setDialer = make(state.SendToStateChan)

	m.openConnection = make(state.SendToStateChan)
	m.addListener = make(state.SendToStateChan)
	m.addErrorListener = make(state.SendToStateChan)
	m.closeTunnel = make(state.SendToStateChan)
	m.close = make(chan bool)

	go m.startStateLoop()
	return m
}

// RunWithTunnel sets the tunnel for the manager, and returns an error if it's already running
func (m *manager) RunWithTunnel(t *tunnel.Tunnel) error {
	return state.InterfaceToError(state.Send(m.setTunnel, t))
}

// RunWithDialer sets the tunnel dialer for the manager, and returns an error if it's already running with a tunnel
func (m *manager) RunWithDialer(d tunnel.Dialer) error {
	return state.InterfaceToError(state.Send(m.setDialer, d))
}

// startStateLoop starts the loop to accept requests over the channels used to synchronously access the manager's state.
// Access the manager's state this way ensures we don't run into deadlocks or race conditions when a tunnel is used for
// both opening and accepting connections.
func (m *manager) startStateLoop() {
	mClosed := false
	for !mClosed {
		ok := true
		var err error
		var dialer tunnel.Dialer
		var tun *tunnel.Tunnel
		var setTunnel, setDialer, closeTunnel, openConnection, addListener, addErrListener state.SendInterface
		var errListeners []chan error
		var tunnelErrs chan struct{}

		for ok {
			if openConnection != nil {
				err = m.handleOpenConnection(tun, openConnection)
			}
			if addListener != nil {
				err = m.handleAddListener(tun, addListener)
			}

			if err != nil {
				if ok = handleError(err, errListeners, tun, dialer); ok == false {
					continue
				}
			}

			if tun != nil {
				tunnelErrs = tun.ErrChan()
			}

			openConnection, addListener, addErrListener, setTunnel, setDialer, err = nil, nil, nil, nil, nil, nil
			select {
			case setTunnel, ok = <-m.setTunnel:
				if !ok {
					continue
				}
				tun = handleSetTunnel(tun, setTunnel)
			case setDialer, ok = <-m.setDialer:
				if !ok {
					continue
				}
				dialer = handleSetDialer(tun, dialer, setDialer)
				tun, err = dialer.Dial()
			case openConnection, ok = <-m.openConnection:
			case addListener, ok = <-m.addListener:
			case addErrListener, ok = <-m.addErrorListener:
				if !ok {
					continue
				}

				errListener := make(chan error)
				errListeners = append(errListeners, errListener)
				addErrListener.Return(errListener)
			case closeTunnel, ok = <-m.closeTunnel:
				if !ok {
					continue
				} else if tun == nil {
					closeTunnel.Return(tunnel.ErrTunnelClosed)
				}

				closeTunnel.Close()
				ok = false
			case <-tunnelErrs:
				err = tun.LastErr
			case <-m.close:
				mClosed = true
				ok = false
			}
		}

		if openConnection != nil {
			openConnection.Return(err)
			openConnection.Close()
		}

		if addListener != nil {
			addListener.Return(err)
			addListener.Close()
		}

		for _, errorListener := range errListeners {
			close(errorListener)
		}

		if tun != nil {
			tun.Close()
		}
	}
}

func writeOutError(listeners []chan error, err error) {
	for _, listener := range listeners {
		select {
		case listener <- err:
		default:
		}
	}
}

func handleError(err error, errListeners []chan error, tun *tunnel.Tunnel, dialer tunnel.Dialer) bool {
	writeOutError(errListeners, err)

	if err == tunnel.ErrTunnelClosed {
		tun = nil
		if dialer == nil {
			return false
		}

		tun, err = dialer.Dial()
		if err != nil {
			writeOutError(errListeners, err)
			return false
		}
	}

	return true
}

func handleSetTunnel(tun *tunnel.Tunnel, setTunnel state.SendInterface) *tunnel.Tunnel {
	defer setTunnel.Close()
	if tun != nil {
		setTunnel.Return(ErrTunnelSet)
	}

	return state.InterfaceToTunnel(setTunnel.Get())
}

func handleSetDialer(tun *tunnel.Tunnel, dialer tunnel.Dialer, setDialer state.SendInterface) tunnel.Dialer {
	defer setDialer.Close()
	if tun != nil || dialer != nil {
		setDialer.Return(ErrTunnelSet)
	}

	return state.InterfaceToDialer(setDialer.Get())
}

// handleOpenConnection is used by the state loop to handle a request to open a connection over the tunnel
func (*manager) handleOpenConnection(tun *tunnel.Tunnel, openConnection state.SendInterface) error {
	if tun == nil {
		openConnection.Return(ErrTunnelNotReady)
		openConnection.Close()
		return nil
	}

	conn, err := tun.Open()
	if err != nil {
		if err == tunnel.ErrTunnelClosed {
			return err
		}

		openConnection.Return(err)
	}

	tlsCfg := state.InterfaceToTLSConfig(openConnection.Get())
	if tlsCfg != nil {
		conn = tls.Client(conn, tlsCfg)
	}

	openConnection.Return(conn)
	openConnection.Close()
	return nil
}

// handleAddListener is used by the request loop to handle a request to retrieve a listener listening over the tunnel
func (m *manager) handleAddListener(tunnel *tunnel.Tunnel, addListener state.SendInterface) error {
	if tunnel == nil {
		addListener.Return(ErrTunnelNotReady)
		addListener.Close()
		return nil
	}

	conResults := make(chan interface{})
	done := tunnel.AcceptWithChannel(conResults)
	addListener.Return(&listener{
		conns: conResults,
		done:  done,
		addr:  tunnel.Addr(),
		close: m.close,
	})

	return nil
}

// Open opens a connection over the tunnel
func (m *manager) Open() (net.Conn, error) {
	return state.InterfaceToConnOrError(state.Send(m.openConnection, nil))
}

// OpenTLS opens a tls connection over the tunnel
func (m *manager) OpenTLS(cfg *tls.Config) (net.Conn, error) {
	return state.InterfaceToConnOrError(state.Send(m.openConnection, cfg))
}

// Listener retrieves a listener listening on the tunnel for connections
func (m *manager) Listener() (net.Listener, error) {
	return state.InterfaceToListenerOrError(state.Send(m.addListener, nil))
}

// ListenForErrors allows the user to register a channel to listen to errors on
func (m *manager) ListenForErrors() chan error {
	return state.InterfaceToErrorChan(state.Send(m.addErrorListener, nil))
}

// CloseTunnel closes the managers tunnel. The tunnel can be reopened with one of the Run functions
func (m *manager) CloseTunnel() error {
	select {
	case <-m.close:
		return ErrManagerClosed
	default:
		return state.InterfaceToError(state.Send(m.closeTunnel, true))
	}
}

// Close closes the manager. A closed manager cannot be reused.
func (m *manager) Close() error {
	m.closeOnce.Do(func() {
		close(m.openConnection)
		close(m.addListener)
		close(m.addErrorListener)

		close(m.closeTunnel)
		close(m.close)
	})

	return nil
}
