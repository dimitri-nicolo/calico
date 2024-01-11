package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/SermoDigital/jose/jwt"
	"github.com/felixge/httpsnoop"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
	_ "k8s.io/component-base/metrics/prometheus/restclient" // to load rest-client metrics

	lmacache "github.com/projectcalico/calico/lma/pkg/cache"
)

const (
	labelHttpMethod     = "method"
	labelSvcAccName     = "svcAcc"
	labelLong           = "long"
	labelStatusCategory = "statusCategory" // e.g. 2xx, 3xx
)

var registerOnce sync.Once

var logger *zap.Logger

func init() {
	// any errors in this package will be repetitive so sample heavily
	loggerCfg := zap.NewProductionConfig()
	loggerCfg.Sampling.Initial = 2
	loggerCfg.Sampling.Thereafter = 0
	root, err := loggerCfg.Build()
	if err != nil {
		log.Fatalf("failed to create zap logger: %v", err)
	}
	logger = root.Named("voltron.metrics")

	// eagerly initialize the metrics by registering them with a throwaway registry
	RegisterMetricsWith(metrics.NewKubeRegistry().MustRegister)
}

var (
	defaultLabelNames = []string{labelHttpMethod, labelLong, labelSvcAccName}

	requestsTotal = metrics.NewCounterVec(&metrics.CounterOpts{
		Name: "http_requests_total",
		Help: "The total number of http requests",
	}, defaultLabelNames)

	httpStatusTotal = metrics.NewCounterVec(&metrics.CounterOpts{
		Name: "http_request_statuses_total",
		Help: "The total number of http requests by status",
	}, []string{labelHttpMethod, labelLong, labelSvcAccName, labelStatusCategory})

	requestTimeSecondsTotal = metrics.NewCounterVec(&metrics.CounterOpts{
		Name: "http_request_time_seconds_total",
		Help: "The total time of http requests in seconds",
	}, defaultLabelNames)

	requestsInflight = metrics.NewGaugeVec(&metrics.GaugeOpts{
		Name: "http_requests_inflight",
		Help: "The current number of requests inflight",
	}, defaultLabelNames)
)

// Metrics used by the inner handler when proxying requests to Linseed.
var (
	innerLabels = []string{"cluster", "tenant", "url"}

	InnerRequestsInflight = metrics.NewGaugeVec(&metrics.GaugeOpts{
		Name: "http_inner_requests_inflight",
		Help: "The current number of requests to Linseed in flight",
	}, innerLabels)

	InnerRequestsTotal = metrics.NewCounterVec(&metrics.CounterOpts{
		Name: "http_inner_requests_total",
		Help: "The total number of http requests to Linseed",
	}, innerLabels)

	InnerRequestTimeSeconds = metrics.NewHistogramVec(&metrics.HistogramOpts{
		Name: "http_inner_request_time_seconds",
		Help: "The duration of http requests to Linseed in seconds",
	}, innerLabels)

	InnerRequestTimeSecondsTotal = metrics.NewCounterVec(&metrics.CounterOpts{
		Name: "http_inner_request_time_seconds_total",
		Help: "The total time of http requests to Linseed in seconds",
	}, innerLabels)

	InnerRequestBadClusterIDErrors = metrics.NewCounterVec(&metrics.CounterOpts{
		Name: "http_inner_request_bad_cluster_id_total",
		Help: "The total number of requests with bad cluster IDs",
	}, innerLabels)

	InnerRequestBadTenantIDErrors = metrics.NewCounterVec(&metrics.CounterOpts{
		Name: "http_inner_request_bad_tenant_id_total",
		Help: "The total number of requests with bad tenant IDs",
	}, innerLabels)
)

func NewHandler() http.Handler {
	registerOnce.Do(func() {
		RegisterMetricsWith(legacyregistry.MustRegister)
		lmacache.RegisterMetricsWith(legacyregistry.MustRegister)
	})

	return promhttp.InstrumentMetricHandler(
		prometheus.DefaultRegisterer,
		promhttp.HandlerFor(legacyregistry.DefaultGatherer, promhttp.HandlerOpts{}),
	)
}

func RegisterMetricsWith(mustRegister func(...metrics.Registerable)) {
	mustRegister(
		requestsTotal,
		requestTimeSecondsTotal,
		requestsInflight,
		httpStatusTotal,
		InnerRequestsInflight,
		InnerRequestsTotal,
		InnerRequestTimeSeconds,
		InnerRequestTimeSecondsTotal,
		InnerRequestBadClusterIDErrors,
		InnerRequestBadTenantIDErrors,
	)
}

func OnRequestStart(r *http.Request, authToken jwt.JWT) func(*httpsnoop.Metrics) {
	long := strings.EqualFold(r.URL.Query().Get("watch"), "true")

	serviceAccountName := "-"
	if authToken != nil {
		if sub, ok := authToken.Claims().Subject(); ok {
			serviceAccountName = toServiceAccountName(sub)
		}
	}

	svcAccLabels := []string{r.Method, strconv.FormatBool(long), serviceAccountName}

	if m, err := requestsTotal.GetMetricWithLabelValues(svcAccLabels...); err != nil {
		logger.Info("failed to get requestsTotal metric", zap.String("name", requestsTotal.Name), zap.Error(err))
	} else {
		m.Inc()
	}

	inflightMetric, inflightErr := requestsInflight.GetMetricWithLabelValues(svcAccLabels...)
	if inflightErr != nil {
		logger.Info("failed to get requestsInflight metric", zap.String("name", requestsTotal.Name), zap.Error(inflightErr))
	} else {
		inflightMetric.Inc()
	}

	return func(snoopMetrics *httpsnoop.Metrics) {
		if inflightErr == nil {
			inflightMetric.Dec()
		}

		if m, err := httpStatusTotal.GetMetricWithLabelValues(
			r.Method,
			strconv.FormatBool(long),
			serviceAccountName,
			httpStatusCategory(snoopMetrics.Code),
		); err != nil {
			logger.Info("failed to get http status metric", zap.String("name", httpStatusTotal.Name), zap.Error(err))
		} else {
			m.Inc()
		}

		if m, err := requestTimeSecondsTotal.GetMetricWithLabelValues(svcAccLabels...); err != nil {
			logger.Info("failed to get requestTimeSecondsTotal metric", zap.String("name", requestsTotal.Name), zap.Error(err))
		} else {
			m.Add(snoopMetrics.Duration.Seconds())
		}
	}
}

func httpStatusCategory(statusCode int) string {
	switch {
	case statusCode == 0: // httpsnoop fails to capture a status sometimes
		return "zero"
	case statusCode < 200:
		return "<200"
	case statusCode < 300:
		return "2XX"
	case statusCode < 400:
		return "3XX"
	case statusCode < 500:
		return "4XX"
	case statusCode < 600:
		return "5XX"
	default:
		return ">599"
	}
}

// toServiceAccountName returns shortened service account names (s:sa:namespace:name) and "non-sa" for anything else
func toServiceAccountName(sub string) string {
	if strings.HasPrefix(sub, "system:serviceaccount:") {
		return strings.ReplaceAll(sub, "system:serviceaccount:", "s:sa:")
	} else {
		return "non-sa"
	}
}
