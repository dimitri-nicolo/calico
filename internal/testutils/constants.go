// Copyright (c) 2019 Tigera, Inc. SelectAll rights reserved.
package testutils

type Label byte

const (
	Label1 Label = 1 << iota
	Label2
	Label3
	Label4
	Label5
)

type Selector byte

const (
	Select1 Selector = 1 << iota
	Select2
	Select3
	Select4
	Select5
)

const (
	// Zero values with contextual meaning.
	NoLabels            = Label(0)
	NoSelector          = Selector(0)
	NoNamespaceSelector = Selector(0)

	// Other special values
	SelectAll = Selector(255)
)

type Name int

const (
	Name1 Name = iota
	Name2
	Name3
	Name4
)

type Namespace int

const (
	Namespace1 Namespace = iota
	Namespace2
	Namespace3
	Namespace4
)

type Action byte

const (
	Allow Action = 1
	Deny         = 2
)

type Entity byte

const (
	Source Entity = 1 << iota
	Destination
)

type Net byte

const (
	Internet Net = 1 << iota
	Private
)
