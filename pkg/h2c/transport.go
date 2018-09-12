package h2c

import (
	"crypto/tls"
	"net"
	"net/http"

	"go.uber.org/zap"

	"golang.org/x/net/http2"
)

// NewTransport will reroute all https traffic to http. This is
// to explicitly allow h2c (http2 without TLS) transport.
// See https://github.com/golang/go/issues/14141 for more details.
func NewTransport() http.RoundTripper {
	return &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {

			return net.Dial(netw, addr)
		},
	}
}

var DefaultTransport http.RoundTripper = NewTransport()

func NewTransportWithLogger(logger *zap.SugaredLogger) http.RoundTripper {
	return &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
			logger.Infof("Recieved connection addr: %s network: %s ", addr, netw)
			conn, err := net.Dial(netw, addr)
			logger.Infof("Finished handling connection remote addr: %s ", conn.RemoteAddr())
			if err != nil {
				logger.Infof("Error handling connection remote addr: %s err: %s ", conn.RemoteAddr, err.Error())
			}
			return conn, err
		},
	}
}
