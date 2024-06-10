package accesslog

import (
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	"github.com/SermoDigital/jose/jwt"
	"github.com/felixge/httpsnoop"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	logger *zap.Logger
	cfg    *config
}

type config struct {
	zapConfig                *zap.Config
	requestHeaders           []fieldMapping
	stringClaims             []fieldMapping
	stringArrayClaims        []fieldMapping
	errorResponseCaptureSize int
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

	logger, err := cfg.zapConfig.Build(
		zap.WithClock(utcSystemClock{}), // always write times in UTC
	)
	if err != nil {
		return nil, err
	}

	return &Logger{
		cfg:    cfg,
		logger: logger,
	}, nil
}

func (l *Logger) OnRequest(w http.ResponseWriter, r *http.Request, authToken jwt.JWT, authTokenError error) (http.ResponseWriter, func(metrics *httpsnoop.Metrics)) {

	// we need to capture these values now, before downstream handlers have a chance to alter them
	tlsField := tlsLogField(r)
	requestField := requestLogField(r, l.cfg, time.Now().UTC(), authToken, authTokenError)

	var capturedResponseBody string

	w = httpsnoop.Wrap(w, httpsnoop.Hooks{
		Write: func(writeFunc httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return func(b []byte) (int, error) {
				errorResponseCaptureSize := l.cfg.errorResponseCaptureSize
				if errorResponseCaptureSize > 0 && capturedResponseBody == "" && b != nil {
					if errorResponseCaptureSize > len(b) {
						errorResponseCaptureSize = len(b)
					}
					capturedResponseBody = string(b[:errorResponseCaptureSize])
				}
				return writeFunc(b)
			}
		},
	})

	return w, func(m *httpsnoop.Metrics) {
		l.logger.Info("",
			tlsField,
			requestField,
			responseLogField(*m, capturedResponseBody),
		)

	}
}

// Flush flushes any log files, primarily required for tests as stderr will be used by default
func (l *Logger) Flush() {
	if l.logger != nil {
		_ = l.logger.Sync() // ignore errors as this will always fail on stderr, but needed for tests which also log to files
	}
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

func requestLogField(r *http.Request, cfg *config, requestTime time.Time, authToken jwt.JWT, authTokenErr error) zap.Field {
	// capture values immediately rather than at the time `ObjectMarshalerFunc` is invoked
	remoteAddr := r.RemoteAddr
	proto := r.Proto
	method := r.Method
	host := r.Host
	path := r.URL.Path
	query := r.URL.RawQuery

	type nv struct {
		name  string
		value string
	}
	var headers []nv // using a slice rather than a map to keep a consistent field order in the output
	for _, h := range cfg.requestHeaders {
		if v := r.Header.Values(h.inputName); len(v) > 0 {
			headers = append(headers, nv{name: h.logFieldName, value: strings.Join(v, "; ")})
		}
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
		for _, h := range headers {
			enc.AddString(h.name, h.value)
		}

		if authTokenErr != nil {
			enc.AddString("authParseFailed", authTokenErr.Error())
		} else if authToken != nil {
			_ = enc.AddObject("auth", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
				encClaimStringField := func(claimName, fieldName string) {
					if value, ok := authToken.Claims().Get(claimName).(string); ok {
						enc.AddString(fieldName, value)
					}
				}
				encClaimStringArrayField := func(claimName, fieldName string) {
					var strArr []string
					if a, ok := authToken.Claims().Get(claimName).([]any); ok {
						for i := range a {
							if s, ok := a[i].(string); ok {
								strArr = append(strArr, s)
							}
						}
					}
					if len(strArr) > 0 {
						_ = enc.AddArray(fieldName, zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
							for i := range strArr {
								enc.AppendString(strArr[i])
							}

							return nil
						}))
					}
				}
				for _, c := range cfg.stringClaims {
					encClaimStringField(c.inputName, c.logFieldName)
				}
				for _, c := range cfg.stringArrayClaims {
					encClaimStringArrayField(c.inputName, c.logFieldName)
				}
				return nil
			}))
		}

		return nil
	}))
}

type utcSystemClock struct{}

func (utcSystemClock) Now() time.Time {
	return time.Now().UTC()
}

func (utcSystemClock) NewTicker(duration time.Duration) *time.Ticker {
	return time.NewTicker(duration)
}
