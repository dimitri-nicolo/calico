package monitor

import (
	bapi "github.com/projectcalico/libcalico-go/lib/backend/api"
	lclient "github.com/tigera/licensing/client"
	"context"
	"time"
	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/apis/v3"
	cerrors "github.com/projectcalico/libcalico-go/lib/errors"
	log "github.com/sirupsen/logrus"
	"reflect"
	"sync"
	"github.com/projectcalico/libcalico-go/lib/set"
)

const (
	defaultPollInterval = 30 * time.Second
)

// LicenseMonitor uses a libcalico-go (backend) client to monitor the status of the active license.
// It provides a thread-safe API for querying the current state of a feature.  Changes to the
// license or its validity are reflected by the API.
type LicenseMonitor struct {
	PollInterval time.Duration
	OnFeaturesChanged func()

	datastoreClient bapi.Client

	activeLicenseLock sync.Mutex
	activeRawLicense *v3.LicenseKey
	activeLicense *lclient.LicenseClaims
}

func New(client bapi.Client) *LicenseMonitor {
	return &LicenseMonitor{
		PollInterval: defaultPollInterval,
		datastoreClient: client,
	}
}

func (l *LicenseMonitor) GetFeatureStatus(feature string) lclient.FeatureStatus {
	l.activeLicenseLock.Lock()
	defer l.activeLicenseLock.Unlock()
	return l.activeLicense.GetFeatureStatus(feature)
}

func (l *LicenseMonitor) MonitorForever(ctx context.Context) error {
	t := jitter.NewTicker(l.PollInterval, l.PollInterval / 10)
	defer t.Stop()

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			break
		case <-t.C:
		}

		l.RefreshLicense()
	}

	return ctx.Err()
}

func (l *LicenseMonitor) RefreshLicense() error {
	log.Debug("Refreshing license from datastore")
	lic, err := l.datastoreClient.Get(context.Background(), model.ResourceKey{
		Kind:      v3.KindLicenseKey,
		Name:      "default",
		Namespace: "",
	}, "")

	l.activeLicenseLock.Lock()
	defer l.activeLicenseLock.Unlock()

	var ttl time.Duration
	oldFeatures := set.New()
	if l.activeLicense != nil {
		ttl = time.Until(l.activeLicense.Expiry.Time())
		oldFeatures = set.FromArray(l.activeLicense.Features)
		log.Debug("Existing license will expire after ", ttl)
	}

	if err != nil {
		switch err.(type) {
		case cerrors.ErrorResourceDoesNotExist:
			if ttl > 0 {
				log.WithError(err).Error("No product license found in the datastore; please contact support; " +
					"already loaded license will expire after ", ttl, " or if component is restarted.")
			} else {
				log.WithError(err).Error("No product license found in the datastore; please install a license " +
					"to enable commercial features.")
			}
			return err
		default:
			if ttl > 0 {
				log.WithError(err).Error("Failed to load product license from datastore; " +
					"already loaded license will expire after ", ttl, " or if component is restarted.")
			} else {
				log.WithError(err).Error("Failed to load product license from datastore.")
			}
			return err
		}
	}

	license := lic.Value.(*v3.LicenseKey)
	log.Debug("License resource found")

	if l.activeRawLicense != nil && reflect.DeepEqual(l.activeRawLicense.Spec, license.Spec) {
		log.Debug("Raw license key data hasn't changed, skipping parse")
		return nil
	}

	newActiveLicense, err := lclient.Decode(*license)
	if err != nil {
		if ttl > 0 {
			log.WithError(err).Error("Failed to decode license key; please contact support; " +
				"already loaded license will expire after ", ttl, " or if component is restarted.")
		} else {
			log.WithError(err).Error("Failed to decode license key; please contact support.")
		}
		return err
	}

	newFeatures := set.FromArray(newActiveLicense.Features)
	if !reflect.DeepEqual(oldFeatures, newFeatures) {
		log.Info("Allowed product features have changed.")
		if l.OnFeaturesChanged != nil {
			l.OnFeaturesChanged()
		}
	}

	l.activeRawLicense = license
	l.activeLicense = &newActiveLicense
	return nil
}
