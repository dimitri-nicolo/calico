// Copyright (c) 2020 Tigera, Inc. All rights reserved.
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

import "time"

func (config *Config) GetFlowLogsPositionFilePath() string {
	return config.WindowsFlowLogsPositionFilePath
}

func (config *Config) GetFlowLogsFileDirectory() string {
	return config.WindowsFlowLogsFileDirectory
}

func (config *Config) GetStatsDumpFilePath() string {
	return config.WindowsStatsDumpFilePath
}

func (config *Config) GetDNSCacheFile() string {
	return config.WindowsDNSCacheFile
}

func (config *Config) GetDNSExtraTTL() time.Duration {
	return config.WindowsDNSExtraTTL
}
