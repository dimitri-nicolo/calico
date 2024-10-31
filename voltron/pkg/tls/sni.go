package tls

import (
	"crypto/tls"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/voltron/pkg/conn"
)

// extractSNI attempts to read the client hello from the TLS Handshake and extract the servername. No bytes are written to
// the connection, and any bytes read from the connection are returned, even if an error occurred.
func extractSNI(connection net.Conn) (string, []byte, error) {
	roConn := conn.NewReadOnly(connection)

	postClientHelloReadStopErr := fmt.Errorf("client hello read, consciously stopping processing")

	var clientHello tls.ClientHelloInfo
	// We use the GetConfigForClient function to hook into the ssl handshake logic and pull out the client hello information,
	// which contains the server name the request is intended for. After this is retrieved, we need to stop tls processing,
	// so we return an error that can be checked against and ignored.
	cfg := &tls.Config{
		GetConfigForClient: func(hi *tls.ClientHelloInfo) (*tls.Config, error) {
			clientHello = *hi

			// Now that we have the client hello we return an error to stop progress on the tls handshake.
			return nil, postClientHelloReadStopErr
		},
	}

	// We use the tls.Server SOLELY to parse the client hello from the connection given. We are not trying to establish or
	// verify a tls connection. This is done so that we don't have to maintain our own client hello parsing logic, and use
	// the well tested parsing logic of the crypto library.
	srv := tls.Server(roConn, cfg)
	defer func() {
		if err := srv.Close(); err != nil {
			log.WithError(err).Error("failed to close tls server")
		}
	}()

	// If there is any error except the expected ErrAttemptedWrite error (which signals we have finished reading the
	// client hello) then return the error
	if err := srv.Handshake(); err != nil && err != postClientHelloReadStopErr {
		return "", roConn.BytesRead(), err
	}

	return clientHello.ServerName, roConn.BytesRead(), nil
}
