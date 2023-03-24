package accesslog

import (
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	"github.com/SermoDigital/jose/jws"
	"github.com/felixge/httpsnoop"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	logger *zap.Logger
	cfg    *config
}

type config struct {
	zapConfig      *zap.Config
	requestHeaders []fieldMapping
	stringClaims   []fieldMapping
}

type fieldMapping struct {
	inputName    string
	logFieldName string
}

var authorizationHeaderName = http.CanonicalHeaderKey("Authorization")
var cookieHeaderName = http.CanonicalHeaderKey("Cookie")

// New returns a zap logger configured specifically for access logging, so no fields such as caller, msg, etc.
func New(options ...Option) (*Logger, error) {
	cfg := &config{
		zapConfig: &zap.Config{
			Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
			Development: false,
			Encoding:    "json",
			EncoderConfig: zapcore.EncoderConfig{
				TimeKey:        "ts",
				NameKey:        "logger",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeTime:     zapcore.ISO8601TimeEncoder,
				EncodeDuration: zapcore.SecondsDurationEncoder,
			},
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
		},
	}

	for _, option := range options {
		if err := option(cfg); err != nil {
			return nil, err
		}
	}

	logger, err := cfg.zapConfig.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{
		cfg:    cfg,
		logger: logger,
	}, nil
}

func (l *Logger) WrapHandler(delegate http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		// we need to capture these values now, before downstream handlers have a chance to alter them
		tlsField := tlsLogField(r)
		requestField := requestLogField(r, l.cfg, time.Now())

		var responseSnippet string

		metrics := httpsnoop.CaptureMetricsFn(w, func(w http.ResponseWriter) {
			// capture the first part of the response which we will log on status >= 400
			w = httpsnoop.Wrap(w, httpsnoop.Hooks{
				Write: func(writeFunc httpsnoop.WriteFunc) httpsnoop.WriteFunc {
					return func(b []byte) (int, error) {
						if responseSnippet == "" && b != nil {
							snippetSize := 250 // the maximum we will take is 250 bytes
							if snippetSize > len(b) {
								snippetSize = len(b)
							}
							responseSnippet = string(b[:snippetSize])
						}
						return writeFunc(b)
					}
				},
			})
			delegate.ServeHTTP(w, r)
		})

		l.logger.Info("",
			tlsField,
			requestField,
			responseLogField(metrics, responseSnippet),
		)
	}
}

// Flush flushes any log files, primarily required for tests as stderr will be used by default
func (l *Logger) Flush() {
	if l.logger != nil {
		_ = l.logger.Sync() // ignore errors as this will always fail on stderr, but needed for tests which also log to files
	}
}

func authorizationHeaderBearerToken(r *http.Request) string {
	if value := r.Header.Get(authorizationHeaderName); len(value) > 7 && strings.EqualFold(value[0:7], "bearer ") {
		return value[7:]
	}
	return ""
}

func tlsLogField(r *http.Request) zap.Field {
	if r.TLS == nil {
		return zap.Skip()
	}

	t := *r.TLS // capture values here rather than at the time `ObjectMarshallerFunc` is invoked

	return zap.Object("tls", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		enc.AddString("proto", t.NegotiatedProtocol)
		enc.AddUint16("version", t.Version)
		enc.AddString("serverName", t.ServerName)
		enc.AddString("cipherSuite", tls.CipherSuiteName(t.CipherSuite))
		return nil
	}))
}

func responseLogField(metrics httpsnoop.Metrics, body string) zap.Field {
	return zap.Object("resp", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		enc.AddInt("status", metrics.Code)
		enc.AddInt64("bytes", metrics.Written)
		enc.AddDuration("duration", metrics.Duration)
		if metrics.Code >= 400 && body != "" {
			enc.AddString("body", body)
		}
		return nil
	}))
}

func requestLogField(r *http.Request, cfg *config, requestTime time.Time) zap.Field {
	// capture values immediately rather than at the time `ObjectMarshalerFunc` is invoked
	remoteAddr := r.RemoteAddr
	proto := r.Proto
	method := r.Method
	host := r.Host
	path := r.URL.Path
	query := r.URL.RawQuery
	xClusterID := r.Header.Get("x-cluster-id")
	userAgent := r.UserAgent()

	var rawAuthToken string
	if len(cfg.stringClaims) > 0 {
		rawAuthToken = authorizationHeaderBearerToken(r)
	}

	return zap.Object("req", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		enc.AddTime("time", requestTime)
		enc.AddString("remoteAddr", remoteAddr)
		enc.AddString("proto", proto)
		enc.AddString("method", method)
		enc.AddString("host", host)
		enc.AddString("path", path)
		if query != "" {
			enc.AddString("query", query)
		}
		if xClusterID != "" {
			enc.AddString("xClusterID", xClusterID)
		}
		if userAgent != "" {
			enc.AddString("userAgent", userAgent)
		}

		if rawAuthToken != "" {
			authToken, err := jws.ParseJWT([]byte(rawAuthToken))
			if err != nil {
				enc.AddString("authParseFailed", err.Error())
			}
			_ = enc.AddObject("auth", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
				encClaimStringField := func(claimName, fieldName string) {
					if value, ok := authToken.Claims().Get(claimName).(string); ok {
						enc.AddString(fieldName, value)
					}
				}
				for _, c := range cfg.stringClaims {
					encClaimStringField(c.inputName, c.logFieldName)
				}
				return nil
			}))
		}

		return nil
	}))
}
