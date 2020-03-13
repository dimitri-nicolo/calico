// Copyright (c) 2019 Tigera Inc. All rights reserved.

package main

import (
	"fmt"
	"reflect"
)

type a struct {
	b *b
}

type b struct {
	i int
}

func main() {
	x := a{&b{1}}
	y := a{&b{1}}
	z := a{&b{2}}
	var n a

	fmt.Println(reflect.DeepEqual(x, y))
	fmt.Println(reflect.DeepEqual(x, z))
	fmt.Println(reflect.DeepEqual(x, n))
	fmt.Println(reflect.DeepEqual(n, n))
}
