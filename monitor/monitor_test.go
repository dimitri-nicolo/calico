package monitor

import (
	"testing"

	. "github.com/onsi/gomega"

	lclient "github.com/tigera/licensing/client"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"context"
	"github.com/projectcalico/libcalico-go/lib/apis/v3"
	"time"
	"errors"
	"gopkg.in/square/go-jose.v2/jwt"
	log "github.com/sirupsen/logrus"
	"sync"
	"sort"
	"github.com/projectcalico/felix/jitter"
)

func TestBasicFunction(t *testing.T) {
	t.Run("With no datastore connection", func(t *testing.T) {
		RegisterTestingT(t)

		// Create a monitor, passing in a nil client.
		m := New(nil)

		// Expect that feature status is returned as false.
		status := m.GetFeatureStatus("foo")
		Expect(status).To(BeFalse())

		// Expect the license to be listed as not loaded.
		Expect(m.GetLicenseStatus()).To(Equal(lclient.NoLicenseLoaded))
	})
}

func TestMonitorLoop(t *testing.T) {
	RegisterTestingT(t)
	m, h := setUpMonitorAndMocks()
	// Increase poll interval and make sure it's out of phase with 30m license expiry time so that we hit the
	// license transition timer.
	m.SetPollInterval(11 * time.Minute)
	defer h.cancel()
	h.license = "good"

	go m.MonitorForever(h.ctx)

	// Wait for the timer to be set up.
	Eventually(h.GetNumTimers).Should(Equal(1))

	// We set PollInterval to 11m and the mock ignores the jitter so the timer should pop at 11m.
	numPops := h.AdvanceTime(10 * time.Minute) // 10m
	Expect(numPops).To(BeZero())
	numPops = h.AdvanceTime(2 * time.Minute)
	Expect(numPops).To(Equal(1)) // 12m
	Eventually(h.GetSignalledLicenseStatus).Should(Equal(lclient.Valid))
	Eventually(h.GetNumTimers).Should(Equal(2))

	numPops = h.AdvanceTime(11 * time.Minute) // 23m
	Expect(numPops).To(Equal(1))

	// Expect the license to go into grace after 30 minutes so jump forward to 29 minutes and check it's still valid...
	numPops = h.AdvanceTime(6 * time.Minute) // 29m
	Consistently(h.GetSignalledLicenseStatus, "100ms", "10ms").Should(Equal(lclient.Valid))

	// Then jump past 30 minutes...
	numPops = h.AdvanceTime(2 * time.Minute)
	Eventually(h.GetSignalledLicenseStatus).Should(Equal(lclient.InGracePeriod))

	// Then jump forward a day, which should end the grace period.
	h.AdvanceTime(23 * time.Hour)
	Consistently(h.GetSignalledLicenseStatus, "100ms", "10ms").Should(Equal(lclient.InGracePeriod))
	h.AdvanceTime(1 * time.Hour)
	Eventually(h.GetSignalledLicenseStatus).Should(Equal(lclient.Expired))
}

func TestRefreshLicense(t *testing.T) {
	t.Run("mainline valid license then expiry test", func(t *testing.T) {
		RegisterTestingT(t)
		m, h := setUpMonitorAndMocks()
		defer h.cancel()
		h.license = "good"

		m.RefreshLicense(h.ctx)
		log.WithField("status", m.GetLicenseStatus()).Info("License status")

		Expect(m.GetLicenseStatus()).To(Equal(lclient.Valid))
		Expect(m.GetFeatureStatus("allowed")).To(BeTrue(), "expected feature to be allowed but it wasn't")
		Expect(m.GetFeatureStatus("foobar")).To(BeFalse(), "expected feature to be disallowed but it wasn't")
		Expect(h.OnFeaturesChangedCalled).To(BeTrue(), "expected feature change to be signalled")
		Expect(m.GetLicenseStatus()).To(Equal(lclient.Valid))

		t.Log("Second call with exactly the same license shouldn't trigger feature change")
		h.OnFeaturesChangedCalled = false
		m.RefreshLicense(h.ctx)
		Expect(h.OnFeaturesChangedCalled).To(BeFalse(), "expected feature change not to be signalled")

		t.Log("After updating license with new features")
		h.license = "good2" // Need to make some tweak to avoid "raw license hasn't changed" optimisation.
		h.allowedFeatures = []string{"some", "new", "features"}
		m.RefreshLicense(h.ctx)
		Expect(h.OnFeaturesChangedCalled).To(BeTrue(), "expected new features to be signalled")

		t.Log("Changing the license without changing the features")
		h.license = "good" // Need to make some tweak to avoid "raw license hasn't changed" optimisation.
		h.OnFeaturesChangedCalled = false
		m.RefreshLicense(h.ctx)
		Expect(h.OnFeaturesChangedCalled).To(BeFalse(), "expected feature change not to be signalled")

		t.Log("changing to a grace-period license")
		h.license = "in-grace"
		m.RefreshLicense(h.ctx)
		Expect(h.OnFeaturesChangedCalled).To(BeFalse(), "expected feature change not to be signalled")
		Expect(m.GetLicenseStatus()).To(Equal(lclient.InGracePeriod))
	})
	t.Run("in grace period", func(t *testing.T) {
		RegisterTestingT(t)
		m, h := setUpMonitorAndMocks()
		defer h.cancel()
		h.license = "in-grace"

		m.RefreshLicense(h.ctx)
		log.WithField("status", m.GetLicenseStatus()).Info("License status")

		Expect(m.GetLicenseStatus()).To(Equal(lclient.InGracePeriod))
		Expect(m.GetFeatureStatus("allowed")).To(BeTrue(), "expected feature to be allowed but it wasn't")
		Expect(m.GetFeatureStatus("foobar")).To(BeFalse(), "expected feature to be disallowed but it wasn't")
	})
	t.Run("with expired license", func(t *testing.T) {
		RegisterTestingT(t)
		m, h := setUpMonitorAndMocks()
		defer h.cancel()
		h.license = "expired"

		m.RefreshLicense(h.ctx)
		log.WithField("status", m.GetLicenseStatus()).Info("License status")

		Expect(m.GetLicenseStatus()).To(Equal(lclient.Expired))
		Expect(m.GetFeatureStatus("allowed")).To(BeFalse(), "expected feature to be allowed but it wasn't")
		Expect(m.GetFeatureStatus("foobar")).To(BeFalse(), "expected feature to be disallowed but it wasn't")
	})
}

func setUpMonitorAndMocks() (*licenseMonitor, *harness) {
	log.SetLevel(log.DebugLevel)
	mockBapiClient := &mockBapiClient{}
	m := New(mockBapiClient).(*licenseMonitor)
	mockTime := &mockTime{
		now: time.Now(), // Start the time epoch now because we can't easily mock the license logic itself.
	}
	m.now = mockTime.Now
	m.newTimer = mockTime.NewTimer
	m.newJitteredTicker = mockTime.NewJitteredTicker
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	h := &harness{
		ctx:             ctx,
		cancel:          cancel,
		mockBapiClient:  mockBapiClient,
		mockTime:        mockTime,
		allowedFeatures: []string{"allowed"},
	}
	m.decodeLicense = h.decodeMockLicense
	m.SetFeaturesChangedCallback(h.OnFeaturesChanged)
	m.SetLicenseStatusChangedCallback(h.OnLicenseStateChanged)
	m.PollInterval = 10 * time.Second
	return m, h
}

type harness struct {
	ctx    context.Context
	cancel context.CancelFunc

	*mockBapiClient
	*mockTime

	allowedFeatures []string

	lock                    sync.Mutex
	OnFeaturesChangedCalled bool
	SignalledLicenseStatus  lclient.LicenseStatus
}

type mockBapiClient struct {
	license string
}

func (m *mockBapiClient) Get(ctx context.Context, key model.Key, revision string) (*model.KVPair, error) {
	return &model.KVPair{Value: &v3.LicenseKey{Spec: v3.LicenseKeySpec{Token: m.license}}}, nil
}

func (h *harness) OnFeaturesChanged() {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.OnFeaturesChangedCalled = true
}

func (h *harness) OnLicenseStateChanged(newLicenseStatus lclient.LicenseStatus) {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.SignalledLicenseStatus = newLicenseStatus
}

func (h *harness) GetSignalledLicenseStatus() lclient.LicenseStatus {
	h.lock.Lock()
	defer h.lock.Unlock()
	return h.SignalledLicenseStatus
}

func (h *harness) decodeMockLicense(lic v3.LicenseKey) (lclient.LicenseClaims, error) {
	log.WithField("raw", lic).Debug("(Mock) decoding license")
	switch lic.Spec.Token {
	case "good", "good2":
		log.Debug("Returning good license")
		return lclient.LicenseClaims{
			Features: h.allowedFeatures,
			Claims: jwt.Claims{
				Expiry: jwt.NewNumericDate(time.Now().Add(30 * time.Minute)),
			},
			GracePeriod: 1,
		}, nil
	case "in-grace":
		log.Debug("Returning grace period license")
		return lclient.LicenseClaims{
			Features: h.allowedFeatures,
			Claims: jwt.Claims{
				Expiry: jwt.NewNumericDate(time.Now().Add(-30 * time.Minute)),
			},
			GracePeriod: 1,
		}, nil
	case "expired":
		log.Debug("Returning expired license")
		return lclient.LicenseClaims{
			Features: h.allowedFeatures,
			Claims: jwt.Claims{
				Expiry: jwt.NewNumericDate(time.Now().Add(-30 * time.Hour)),
			},
			GracePeriod: 1,
		}, nil
	}
	return lclient.LicenseClaims{}, errors.New("bad license")
}

type mockTime struct {
	lock sync.Mutex
	now        time.Time
	timerQueue []*queueEntry
}

type queueEntry struct {
	PopTime time.Time
	Timer   *time.Timer
	Ticker  *jitter.Ticker
	Duration time.Duration
	Stopped chan struct{}
	C chan time.Time
}

func (q *queueEntry) Stop() bool {
	close(q.Stopped)
	return false
}

func (q *queueEntry) Chan() <-chan time.Time {
	return q.C
}

func (t *mockTime) Now() time.Time {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.now
}

func (t *mockTime) NewTimer(d time.Duration) timer {
	t.lock.Lock()
	defer t.lock.Unlock()
	c := make(chan time.Time)
	timer := &time.Timer{C: c}
	queueEntry := queueEntry{
		PopTime: t.now.Add(d),
		Timer:   timer,
		C: c,
		Stopped: make(chan struct{}),
	}
	t.timerQueue = append(t.timerQueue, &queueEntry)
	sort.Slice(t.timerQueue, func(i, j int) bool {
		return t.timerQueue[i].PopTime.Before(t.timerQueue[j].PopTime)
	})
	return &queueEntry
}

func (t *mockTime) NewJitteredTicker(d time.Duration, jit time.Duration) *jitter.Ticker {
	t.lock.Lock()
	defer t.lock.Unlock()

	c := make(chan time.Time)
	timer := &jitter.Ticker{C: c}
	queueEntry := queueEntry{
		PopTime: t.now.Add(d),
		Ticker: timer,
		Duration: d,
		C: c,
		Stopped: make(chan struct{}),
	}
	t.timerQueue = append(t.timerQueue, &queueEntry)
	return timer
}

func (t *mockTime) AdvanceTime(d time.Duration) int {
	newTime := t.now.Add(d)
	log.Info("Advancing time to ", newTime)
	numPops := 0

	for {
		t.lock.Lock()
		if len(t.timerQueue) == 0 {
			// No timers left...
			t.lock.Unlock()
			break
		}
		t.sortQueue()
		firstTimer := t.timerQueue[0]
		if firstTimer.PopTime.After(newTime) {
			// Timer is in the future so there's nothing to do.
			t.lock.Unlock()
			break
		}
		t.now = firstTimer.PopTime
		t.timerQueue = t.timerQueue[1:]
		t.lock.Unlock()

		// Can't hold the lock while we pop the timer or we might deadlock with the code under test scheduling a new
		// one.
		select {
		case firstTimer.C <- firstTimer.PopTime:
			numPops++
		case <-firstTimer.Stopped:
			continue
		}

		if firstTimer.Ticker != nil {
			// This is a ticker, reschedule it.
			firstTimer.PopTime = firstTimer.PopTime.Add(firstTimer.Duration)
			t.lock.Lock()
			t.timerQueue = append(t.timerQueue, firstTimer)
			t.lock.Unlock()
		}
	}

	t.now = newTime
	return numPops
}

func (t *mockTime) sortQueue() {
	sort.Slice(t.timerQueue, func(i, j int) bool {
		return t.timerQueue[i].PopTime.Before(t.timerQueue[j].PopTime)
	})
}

func (t *mockTime) GetNumTimers( ) int{
	t.lock.Lock()
	defer t.lock.Unlock()
	return len(t.timerQueue)
}
