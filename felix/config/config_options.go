// Copyright (c) 2020-2023 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	log "github.com/sirupsen/logrus"
)

type Option func(c *Config)

func WithDNSDoNotWriteIPSets() Option {
	return func(c *Config) {
		_, _ = c.OverrideParam("DNSDoNotWriteIPSets", "true")
		log.Infof("Set config DNSDoNotWriteIPSets=true")
	}
}
