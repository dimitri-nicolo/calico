package testutils

import (
	envoycore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyauthz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CheckRequestBuilder struct {
	srcHost, dstHost                     string
	srcPort, dstPort                     uint32
	protocol, method, scheme, host, path string
	headers                              map[string]string
}

type CheckRequestBuilderOption func(*CheckRequestBuilder)

func NewCheckRequestBuilder(opts ...CheckRequestBuilderOption) *CheckRequestBuilder {
	res := &CheckRequestBuilder{
		srcHost:  "0.0.0.0",
		srcPort:  0,
		dstHost:  "127.0.0.1",
		dstPort:  80,
		protocol: "HTTP/1.1",
		method:   "GET",
		scheme:   "http",
		host:     "www.example.com",
		path:     "/",
		headers:  map[string]string{},
	}

	for _, opt := range opts {
		opt(res)
	}
	return res
}

func WithSourceHostPort(srcHost string, srcPort uint32) CheckRequestBuilderOption {
	return func(b *CheckRequestBuilder) {
		b.srcHost = srcHost
		b.srcPort = srcPort
	}
}

func WithDestinationHostPort(dstHost string, dstPort uint32) CheckRequestBuilderOption {
	return func(b *CheckRequestBuilder) {
		b.dstHost = dstHost
		b.dstPort = dstPort
	}
}

func WithMethod(method string) CheckRequestBuilderOption {
	return func(b *CheckRequestBuilder) {
		b.method = method
	}
}

func WithHost(host string) CheckRequestBuilderOption {
	return func(b *CheckRequestBuilder) {
		b.host = host
	}
}

func WithPath(path string) CheckRequestBuilderOption {
	return func(b *CheckRequestBuilder) {
		b.path = path
	}
}

func WithScheme(scheme string) CheckRequestBuilderOption {
	return func(b *CheckRequestBuilder) {
		b.scheme = scheme
	}
}

func (b *CheckRequestBuilder) Value() *envoyauthz.CheckRequest {
	return &envoyauthz.CheckRequest{
		Attributes: &envoyauthz.AttributeContext{
			Source: &envoyauthz.AttributeContext_Peer{
				Address: addressFromHostPort(b.srcHost, b.srcPort),
			},
			Destination: &envoyauthz.AttributeContext_Peer{
				Address: addressFromHostPort(b.dstHost, b.dstPort),
			},
			Request: &envoyauthz.AttributeContext_Request{
				Time: timestamppb.Now(),
				Http: &envoyauthz.AttributeContext_HttpRequest{
					Headers:  b.headers,
					Protocol: b.protocol,
					Method:   b.method,
					Scheme:   b.scheme,
					Host:     b.host,
					Path:     b.path,
				},
			},
		},
	}
}

func addressFromHostPort(host string, port uint32) *envoycore.Address {
	return &envoycore.Address{
		Address: &envoycore.Address_SocketAddress{
			SocketAddress: &envoycore.SocketAddress{
				Address: host,
				PortSpecifier: &envoycore.SocketAddress_PortValue{
					PortValue: port,
				},
			},
		},
	}
}
