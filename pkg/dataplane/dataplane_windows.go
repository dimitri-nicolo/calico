// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.
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

package dataplane

import (
<<<<<<< HEAD
	"github.com/sirupsen/logrus"

	"github.com/projectcalico/cni-plugin/pkg/dataplane/windows"
	"github.com/projectcalico/cni-plugin/pkg/types"
=======
	"github.com/projectcalico/cni-plugin/pkg/dataplane/windows"
	"github.com/projectcalico/cni-plugin/pkg/types"
	"github.com/sirupsen/logrus"
>>>>>>> a8ef39ddee9917a87d27d74f1d47079158f595c9
)

func getDefaultSystemDataplane(conf types.NetConf, logger *logrus.Entry) (Dataplane, error) {
	return windows.NewWindowsDataplane(conf, logger), nil
}
