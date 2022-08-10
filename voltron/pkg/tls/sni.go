package tls

import (
	"crypto/tls"
	"net"

	calicotls "github.com/projectcalico/calico/crypto/pkg/tls"
	"github.com/projectcalico/calico/voltron/pkg/conn"

	log "github.com/sirupsen/logrus"
)

// placeholder certificate used by the TLS server that extracts the SNI from the request. Common name is for sni.extraction.placeholder.
var certBytes = []byte(`-----BEGIN CERTIFICATE-----
MIIDtTCCAp2gAwIBAgIUT2CuGwvBxGoTK+kqq0w8nNohXRQwDQYJKoZIhvcNAQEL
BQAwajELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDEjMCEGA1UEAwwac25pLmV4dHJhY3Rp
b24ucGxhY2Vob2xkZXIwHhcNMjAwMjIxMDA1OTU0WhcNMjAwMzIyMDA1OTU0WjBq
MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMSMwIQYDVQQDDBpzbmkuZXh0cmFjdGlvbi5w
bGFjZWhvbGRlcjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAPSrg+d3
HEPOXmHsmwtSHO06B+Te5sFzfRxCmpPE5o3mjzKYv9XL3NRpko6ONsHfTHc39lzg
KzOEbTDUDRJh095eyo4+k3IMGs+xU0K6QJdAPYazlK51NGj0U4Bc74Ky06OkusLP
71LYKsVJRE0aqgOpBSYI2zIGofKJfpMHNly9FKvruxM+LMpdknjTs0NfeI1IrmdW
UlMon7D3XnHlyd7q+LynAxbBw+VV6395LaCyHTuZ3YtbVHqPFFEwnRBSOe0e4f5B
z3NzQO29BQpFlc1erXrIs9TuN/ZRWHEsJsQXys5yOsSHLrYjzwo4hpLOYd7u9MMW
zdErEZGU/oJcCAcCAwEAAaNTMFEwHQYDVR0OBBYEFO+0BGUy8M1490dIY8Gn++wK
3D7BMB8GA1UdIwQYMBaAFO+0BGUy8M1490dIY8Gn++wK3D7BMA8GA1UdEwEB/wQF
MAMBAf8wDQYJKoZIhvcNAQELBQADggEBAJMWkmoWtywXoYljkcVt4h5xx1lAC5xa
4zM607czF+Wgm7iuUVOA/YVYzBKlmdLL3nwbax2+eSwhi4hKzjFAbAjnsTAloXvl
x1ilpCRs6pbxhVU0Q+6CANzNQ7F+OjJ379Gvhb2uW9xhgBf0YimQ1msHvHW8FTtu
1N9FA2HAMm6qI6/cS9fcozKBthTRQoP3yvBqoc4SfzhSszc7MeEjDJUNAH0TFMwJ
y4xB9b7afE7ZArHSsrtr3tdbKAn8MYNE3UZVn3dv18mXMoR53uGWmtkxiX7XSAcY
9ShtVA1q110MZoQEcU+nd1Ix8FXWH//ur5/gty7Y3t69OjXa7rK6mL4=
-----END CERTIFICATE-----`)

// placeholder key used by the TLS server that extracts the SNI from the request.
var keyBytes = []byte(`-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQD0q4PndxxDzl5h
7JsLUhztOgfk3ubBc30cQpqTxOaN5o8ymL/Vy9zUaZKOjjbB30x3N/Zc4CszhG0w
1A0SYdPeXsqOPpNyDBrPsVNCukCXQD2Gs5SudTRo9FOAXO+CstOjpLrCz+9S2CrF
SURNGqoDqQUmCNsyBqHyiX6TBzZcvRSr67sTPizKXZJ407NDX3iNSK5nVlJTKJ+w
915x5cne6vi8pwMWwcPlVet/eS2gsh07md2LW1R6jxRRMJ0QUjntHuH+Qc9zc0Dt
vQUKRZXNXq16yLPU7jf2UVhxLCbEF8rOcjrEhy62I88KOIaSzmHe7vTDFs3RKxGR
lP6CXAgHAgMBAAECggEAHiSLSZbpComAIzxNFaX2HlvJ4S5861RZE4Q5Gv9lEBJZ
jfg3mhVVjW28OofWwyfJed6RIXwUlnI4KY3WVm9q9Lhk6AVZkPFg1DmaclwT3Q5z
BgdVx/B0loGTT/sjHsz9OenvgFSxvVkYW9nc6krgqzbFhZwNtSoQBZte1qpKzj3X
hP/RAOW7H9I9KjEsZ2xBFW/HO+u+SVPLB1m3gd0ehYmV7L2jNBp0OyF5akHszuF5
TUqOvU1kN5op9WDH0peFH3th/I4rXjWbQ9f75XH2c4fYqnV/uLO7IE3OH7m/+9pB
6N6Hs6g7FghGlo/PkEp43l1tMQdojNpZ1OKmri0mcQKBgQD8Rp38Wp1oTH8cBNQf
hpxMjU5uxgZ5xQczIvlzHgiw19qz9BPK3CJVbyjgVhuNoq6UWiDTkKKI4YcS07eP
J6tuP9sB6apTMttiYvspVKQJS+Mjsm/O0hl+ZUJmOh3kIbUQ/Hvxx9nmI5C9eMlX
pQ75g2fbJto7mmE4p0jowb1ebQKBgQD4SCeR7b7C66hnOGEGTncvwEh6vbCr8Q2Q
eUujDLGluJ4kPfea2FfDgx8jhVYLhA+T07dv02bA6a5aaQfCh7aYN32I881TZu1u
QRg8TmYBxfWqjsM7sNrRaLKqgmUVHbeUn8nAMsVLKgam0Ra3tU8Ali0A7ZO27A8S
tYWdXhSnwwKBgA4F8uRDOTrB/dLV5eC2v9t1g2We9l8wd5z9FbazdbI23X5hU/RT
1ki/fBs0TiXKZD/03pxEDvTi7Ho8cJixkNL5E7iAf6pOSmmmrOV4QgIOSNsEITjy
7t3azR0Xn++9e+4sysr+2/ryASq3GyIXF8UA6/X/q+PiSgM3MVNW6arlAoGAIBd7
feJEEP/S4Zyo9d64iySIec0BBAiBX1Y+T5H5eFk3n6me0pX6KhxNrxKx/4UPWmU4
Ra0GkBLkZW1EAoH2ORCbGlOhC5G3SNQDJPBhQQNscKJJW/LNJdopld6K4ELaEszg
kAY/+Cozd+Z40EAQORwwLvmGaVNz01BBOAkMFG8CgYEAy4FsKDcO2PT0QusUyVJD
yoEbEVnUyjU0h5gBdzqWWbqZW9EfCKr5d6E30dy8GCavA47ZzncczTMAzZUAJx3t
oEWtInAu/pT/O1/RtYzgCopG6Kg6rqrUjTnC1HXd1HXSBJjW9ORBmoNJto2LL9tb
Zjr9rbhG52zGe3CnwwPUWH8=
-----END PRIVATE KEY-----`)

// extractSNI attempts to read the client hello of the TLS Handshake and extract the servername. No bytes are written to
// the connection, and any bytes read from the connection are returned, even if an error occurred.
func extractSNI(connection net.Conn, fipsModeEnabled bool) (string, []byte, error) {
	roConn := conn.NewReadOnly(connection)

	// we need to provide the tls certificates to avoid returning a "no tls certificates found" error in the server
	cert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		return "", nil, err
	}

	cfg := calicotls.NewTLSConfig(fipsModeEnabled)
	cfg.Certificates = []tls.Certificate{cert}
	srv := tls.Server(roConn, cfg)
	defer func() {
		if err := srv.Close(); err != nil {
			log.WithError(err).Error("failed to close tls server")
		}
	}()

	// If there is any error except the expected ErrAttemptedWrite error (which signals we have finished reading the
	// client hello) then return the error
	if err := srv.Handshake(); err != nil && err != conn.ErrAttemptedWrite {
		return "", roConn.BytesRead(), err
	}

	return srv.ConnectionState().ServerName, roConn.BytesRead(), nil
}
