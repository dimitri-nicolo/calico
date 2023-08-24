//go:build !cgo
// +build !cgo

// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.
package waf

import "github.com/google/uuid"

func IsEnabled() bool {
	return false
}

func CheckRulesSetExists(_ string) error {
	return nil
}

func Initialize(_ interface{}) {
}

func InitializeModSecurity() {
}

func GetRulesSetFilenames() []string {
	return nil
}

func LoadModSecurityCoreRuleSet(_ []string) error {
	return nil
}

func GenerateModSecurityID() string {
	return uuid.New().String()
}

func ProcessHttpRequest(_, _, _, _, _, _ string, _ uint32, _ string, _ uint32, _ map[string]string, _ string) error {
	return nil
}

func GetAndClearOwaspLogs(_ string) []*OwaspInfo {
	return nil
}
func GetProcessHttpRequestPrefix(_ string) string {
	return ""
}

func CleanupModSecurity() {}
