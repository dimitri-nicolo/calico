package server

import (
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/voltron/internal/pkg/proxy"
)

// listenAndForward accepts connections on the given listener and forwards them to the given service
func listenAndForward(listener net.Listener, serverName string, retryAttempts int, retryInterval time.Duration) error {
	for {
		srcConn, err := listener.Accept()
		if err != nil {
			return err
		}

		dstConn, err := dialServer(serverName, retryAttempts, retryInterval)
		if err != nil {
			log.WithError(err).Errorf("failed to open a connection proxy service %s", serverName)
			if err := srcConn.Close(); err != nil {
				log.WithError(err).Error("failed to close source connection")
			}
			continue
		}

		go proxy.ForwardConnection(srcConn, dstConn)
	}
}

func dialServer(serverName string, retryAttempts int, retryInterval time.Duration) (net.Conn, error) {
	var dstConn net.Conn
	var err error
	for i := 0; i < retryAttempts; i++ {
		var err error
		dstConn, err = net.Dial("tcp", serverName)
		if err != nil {
			log.WithError(err).Debugf("failed to open a connection to %s, will retry in %d seconds", serverName, retryInterval)
			time.Sleep(retryInterval * time.Second)
			continue
		}

		break
	}

	return dstConn, err
}
