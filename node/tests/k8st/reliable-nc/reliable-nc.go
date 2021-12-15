// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

func main() {
	destination := os.Args[1]
	if strings.Contains(destination, ":") {
		// Client.
		_, _, err := net.SplitHostPort(destination)
		if err != nil {
			fmt.Printf("ERROR: arg '%v' fails SplitHostPort: %v\n", destination, err)
			os.Exit(-1)
		}
		conn, err := net.Dial("tcp", destination)
		if err != nil {
			fmt.Printf("ERROR: failed to establish TCP connection to %v: %v\n", destination, err)
			os.Exit(-1)
		}
		written, err := io.Copy(conn, os.Stdin)
		if err != nil {
			fmt.Printf("ERROR: failed to write data to TCP connection to %v: %v\n", destination, err)
			os.Exit(-1)
		}
		fmt.Printf("Success, copied %v bytes to %v\n", written, destination)
	} else {
		// Server.
		listener, err := net.Listen("tcp", "0.0.0.0:"+destination)
		if err != nil {
			fmt.Printf("ERROR: failed to start listen on TCP port %v: %v\n", destination, err)
			os.Exit(-1)
		}
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("ERROR: failed to accept incoming connection on TCP port %v: %v\n", destination, err)
			os.Exit(-1)
		}
		read, err := io.Copy(os.Stdout, conn)
		if err != nil {
			fmt.Printf("ERROR: failed to read data from TCP connection on port %v: %v\n", destination, err)
			os.Exit(-1)
		}
		fmt.Printf("Success, read %v bytes on TCP port %v\n", read, destination)
	}
	os.Exit(0)
}
