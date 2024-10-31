// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package fileutils

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/deep-packet-inspection/pkg/weputils"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
)

// alertFileRelativePath generates the path of alert file relative to the alert base path.
// relative alert path is `dpi_namespace/dpi_name/wep_podName`
func AlertFileRelativePath(dpiKey model.ResourceKey, wepKey model.WorkloadEndpointKey) string {
	_, podName, err := weputils.ExtractNamespaceAndNameFromWepName(wepKey.WorkloadID)
	if err != nil {
		log.WithError(err).Error("Failed to get pod name from WEP key")
	}
	return fmt.Sprintf("%s/%s/%s", dpiKey.Namespace, dpiKey.Name, podName)
}

// alertFileAbsolutePath generates absolute path of alert file.
// absolute path is `file_base_path_from_config/dpi_namespace/dpi_name/wep_podName`
func AlertFileAbsolutePath(dpiKey model.ResourceKey, wepKey model.WorkloadEndpointKey, SnortAlertFileBasePath string) string {
	return fmt.Sprintf("%s/%s", SnortAlertFileBasePath, AlertFileRelativePath(dpiKey, wepKey))
}
