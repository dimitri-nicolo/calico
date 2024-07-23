// Copyright (c) 2018-2024 Tigera, Inc. All rights reserved.

package uds

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

func init() {
	resolver.SetDefaultScheme("passthrough")
}

func getDialer(proto string) func(context.Context, string) (net.Conn, error) {
	d := &net.Dialer{}
	return func(ctx context.Context, target string) (net.Conn, error) {
		return d.DialContext(ctx, proto, target)
	}
}

func GetDialOptions() []grpc.DialOption {
	return GetDialOptionsWithNetwork("unix")
}

func GetDialOptionsWithNetwork(network string) []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(getDialer(network))}
}
