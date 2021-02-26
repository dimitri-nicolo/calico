// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", os.Args[1])
	if err != nil {
		fmt.Printf("ERROR: failed to establish TCP connection to %v: %v\n", os.Args[1], err)
		os.Exit(-1)
	}
	written, err := io.Copy(conn, os.Stdin)
	if err != nil {
		fmt.Printf("ERROR: failed to write data to TCP connection to %v: %v\n", os.Args[1], err)
		os.Exit(-1)
	}
	fmt.Printf("Success, copied %v bytes to %v\n", written, os.Args[1])
	os.Exit(0)
}
