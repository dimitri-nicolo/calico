// Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.

package uds

import (
	"context"
	"net"

	"google.golang.org/grpc"
)

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
		grpc.WithInsecure(),
		grpc.WithContextDialer(getDialer(network))}
}
