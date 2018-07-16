package monitor

import (
	"context"
	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/libcalico-go/lib/apis/v3"
	bapi "github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	cerrors "github.com/projectcalico/libcalico-go/lib/errors"
	"github.com/projectcalico/libcalico-go/lib/set"
	log "github.com/sirupsen/logrus"
	lclient "github.com/tigera/licensing/client"
	"reflect"
	"sync"
	"time"
)

const (
	defaultPollInterval = 30 * time.Second
)

// LicenseMonitor is an interface which enables monitoring of license and feature enablement status.
type LicenseMonitor interface {
	GetFeatureStatus(string) bool
	GetLicenseStatus() lclient.LicenseStatus
	MonitorForever(context.Context) error
	RefreshLicense(context.Context) error
	SetFeaturesChangedCallback(func())
}

// licenseMonitor uses a libcalico-go (backend) client to monitor the status of the active license.
// It provides a thread-safe API for querying the current state of a feature.  Changes to the
// license or its validity are reflected by the API.
type licenseMonitor struct {
	PollInterval      time.Duration
	OnFeaturesChanged func()

	datastoreClient bapi.Client

	activeLicenseLock sync.Mutex
	activeRawLicense  *v3.LicenseKey
	activeLicense     *lclient.LicenseClaims
}

func New(client bapi.Client) LicenseMonitor {
	return &licenseMonitor{
		PollInterval:    defaultPollInterval,
		datastoreClient: client,
	}
}

func (l *licenseMonitor) GetFeatureStatus(feature string) bool {
	l.activeLicenseLock.Lock()
	defer l.activeLicenseLock.Unlock()
	return l.activeLicense.ValidateFeature(feature)
}

func (l *licenseMonitor) GetLicenseStatus() lclient.LicenseStatus {
	l.activeLicenseLock.Lock()
	defer l.activeLicenseLock.Unlock()
	return l.activeLicense.Validate()
}

func (l *licenseMonitor) SetFeaturesChangedCallback(f func()) {
	l.OnFeaturesChanged = f
}

func (l *licenseMonitor) MonitorForever(ctx context.Context) error {
	// TODO: use jitter package in libcalico-go once it has been ported to
	// libcalico-go-private.
	t := jitter.NewTicker(l.PollInterval, l.PollInterval/10)
	defer t.Stop()

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			break
		case <-t.C:
		}

		l.RefreshLicense(ctx)
	}

	return ctx.Err()
}

func (l *licenseMonitor) RefreshLicense(ctx context.Context) error {
	log.Debug("Refreshing license from datastore")
	lic, err := l.datastoreClient.Get(ctx, model.ResourceKey{
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
				log.WithError(err).Error("No product license found in the datastore; please contact support; "+
					"already loaded license will expire after ", ttl, " or if component is restarted.")
			} else {
				log.WithError(err).Error("No product license found in the datastore; please install a license " +
					"to enable commercial features.")
			}
			return err
		default:
			if ttl > 0 {
				log.WithError(err).Error("Failed to load product license from datastore; "+
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
			log.WithError(err).Error("Failed to decode license key; please contact support; "+
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
