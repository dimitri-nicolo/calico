// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Package regex has a set of regex related functions used across components
package regex

import (
	"reflect"
	"regexp"
	"testing"
)

func TestCompileRegexStrings(t *testing.T) {
	patternStr := `^/compliance/?`
	expectedRegexp, err := regexp.Compile(patternStr)
	if err != nil {
		t.Errorf("TestCompileRegexStrings failed to compile expectedRegexp: %s", err)
	}

	type args struct {
		patterns []string
	}
	tests := []struct {
		name    string
		args    args
		want    []regexp.Regexp
		wantErr bool
	}{
		{
			name:    "Should return an empty list of Regexp",
			args:    args{patterns: []string{}},
			want:    []regexp.Regexp{},
			wantErr: false,
		},
		{
			name:    "Should return a list of one Regexp",
			args:    args{patterns: []string{patternStr}},
			want:    []regexp.Regexp{*expectedRegexp},
			wantErr: false,
		},
		{
			name:    "Should return an error (invalid regex string)",
			args:    args{patterns: []string{`?!`}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompileRegexStrings(tt.args.patterns)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileRegexStrings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CompileRegexStrings() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
