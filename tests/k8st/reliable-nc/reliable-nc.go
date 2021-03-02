// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	destination := os.Args[1]
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
	os.Exit(0)
}
