// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package app

import "embed"

//go:embed template-docker/*
var DockerTemplates embed.FS

//go:embed template-github/*
var GithubTemplates embed.FS

//go:embed template-http-server/*
var HTTPServerTemplates embed.FS

//go:embed template-proxy/*
var ProxyTemplates embed.FS

//go:embed template-makefile/*
var MakefileTemplates embed.FS

//go:embed template-semaphore/*
var SemaphoreTemplates embed.FS

var GoServerTemplates = map[embed.FS]string{
	DockerTemplates:     "template-docker",
	GithubTemplates:     "template-github",
	HTTPServerTemplates: "template-http-server",
	MakefileTemplates:   "template-makefile",
	SemaphoreTemplates:  "template-semaphore",
}

var GoProxyTemplates = map[embed.FS]string{
	DockerTemplates:    "template-docker",
	GithubTemplates:    "template-github",
	ProxyTemplates:     "template-proxy",
	MakefileTemplates:  "template-makefile",
	SemaphoreTemplates: "template-semaphore",
}

var SkeletonTemplates = map[embed.FS]string{
	DockerTemplates:    "template-docker",
	GithubTemplates:    "template-github",
	MakefileTemplates:  "template-makefile",
	SemaphoreTemplates: "template-semaphore",
}
