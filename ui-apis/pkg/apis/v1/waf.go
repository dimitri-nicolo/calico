// Copyright (c) 2025 Tigera, Inc. All rights reserved.
package v1

type WAFRuleset struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Files []File `json:"files"`
}

type File struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Rules []Rule `json:"rules"`
}

type Rule struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Data string `json:"data,omitempty"`
}
