package cis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/aquasecurity/kube-bench/check"
	"github.com/coreos/go-semver/semver"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"

	"github.com/tigera/compliance/pkg/benchmark"
	"github.com/tigera/compliance/pkg/datastore"
	api "github.com/tigera/lma/pkg/api"
)

const (
	leastKubeBenchSupportedVersion = "1.6.0"
	patchNumber                    = "0"
)

// Benchmarker implements benchmark.Executor
type Benchmarker struct {
	ConfigChecker func(string) bool
}

// NewBenchmarker returns a benchmark.Executor instance that can execute kubernetes cis benchmark tests
func NewBenchmarker() api.BenchmarksExecutor {
	return &Benchmarker{ConfigChecker: configExists}
}

// ExecuteBenchmarks determines the appropriate benchmarker to run for the given benchmark type.
func (b *Benchmarker) ExecuteBenchmarks(ctx context.Context, ct api.BenchmarkType, nodename string) (*api.Benchmarks, error) {
	if ct == api.TypeKubernetes {
		return b.executeKubeBenchmark(ctx, nodename)
	}
	return nil, fmt.Errorf("No handler found for benchmark type %s", ct)
}

func configExists(cfgPath string) bool {
	_, err := os.Stat(cfgPath)
	return !os.IsNotExist(err)
}

// Get the version for which a corresponding configuration exists.
// e.g. 1.12 matches 1.11
func (b *Benchmarker) GetClosestConfig(dv string) (string, error) {
	// kube-bench CIS benchmark version starts from k8s version 1.6
	//   -- https://github.com/aquasecurity/kube-bench#cis-kubernetes-benchmark-support
	kubeBenchBaseVersion, err := semver.NewVersion(leastKubeBenchSupportedVersion)
	if err != nil {
		return "", err
	}

	detectedVersion, err := semver.NewVersion(dv)
	if err != nil {
		return "", err
	}
	if detectedVersion.LessThan(*kubeBenchBaseVersion) {
		return "", fmt.Errorf("CIS Kubernetes Benchmark doesn't support kubernetes version < v1.6")
	}

	// starting from the base version, bump up to the detected version, return
	// the last version for which a configuration exists.
	resultVersion := *kubeBenchBaseVersion
	tempVersion := kubeBenchBaseVersion
	for detectedVersion.String() != tempVersion.String() {
		tempVersion.BumpMinor()
		// Check if the matching kube-bench config exists.
		if b.ConfigChecker(fmt.Sprintf("/opt/cfg/%d.%d", tempVersion.Major, tempVersion.Minor)) {
			resultVersion = *tempVersion
			fmt.Printf("%v", resultVersion)
		}
	}
	return fmt.Sprintf("%d.%d", resultVersion.Major, resultVersion.Minor), nil
}

func (b *Benchmarker) getClosestKubeBenchConfigVersion() (closestVersion string, err error) {
	// Get k8s server version.
	restConfig := datastore.MustGetConfig()
	discovery, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		log.WithError(err).Debug("error getting kubernetes server version")
		return closestVersion, err
	}
	k8sversion, err := discovery.ServerVersion()
	if err != nil {
		log.WithError(err).Debug("error getting kubernetes server version")
		return closestVersion, err
	}

	detectedVersion := fmt.Sprintf("%s.%s.%s", k8sversion.Major, k8sversion.Minor, patchNumber)
	log.Debug("server version received: ", detectedVersion)
	detectedVersionFormatted := b.GetSemVerFormatted(detectedVersion)
	log.Debug("server version formatted: ", detectedVersionFormatted)
	closestVersion, err = b.GetClosestConfig(detectedVersionFormatted)
	if err != nil {
		return closestVersion, err
	}

	return
}

func (b *Benchmarker) GetSemVerFormatted(v string) string {
	semVerRegExp := regexp.MustCompile(`(\d+).*?(\d+).+?(\d+).*`)
	matches := semVerRegExp.FindStringSubmatch(v)
	if len(matches) > 3 {
		return fmt.Sprintf("%s.%s.%s", matches[1], matches[2], matches[3])
	}

	return leastKubeBenchSupportedVersion
}

// executeKubeBenchmark executes kube-bench.
func (b *Benchmarker) executeKubeBenchmark(ctx context.Context, nodename string) (*api.Benchmarks, error) {
	// Determine Openshift args if any.
	args, err := determineOpenshiftArgs(nodename)
	if err != nil {
		return nil, err
	}

	// If not running OCP, get kube-bench config version.
	if len(args) == 0 {
		version, err := b.getClosestKubeBenchConfigVersion()
		if err != nil {
			return nil, err
		}
		args = append(args, "--version", version)
	}

	args = append(args, "--json")
	log.WithField("cmd", args).Debug("executing benchmarker")

	// Execute the benchmarker
	ts := time.Now()
	cmd := exec.Command("kube-bench", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.WithField("output", string(out)).WithError(err).Error("Failed to execute kubernetes benchmarker")
		return nil, err
	}
	res := bytes.Split(out, []byte("\n"))

	// Parse benchmarker results
	ctrls := []*check.Controls{}
	for _, line := range res {
		ctrl := new(check.Controls)
		if err := json.Unmarshal(line, ctrl); err != nil {
			fmt.Println("failed to unmarshal json", string(line), err)
			continue
		}
		ctrls = append(ctrls, ctrl)
	}
	if len(ctrls) == 0 {
		return nil, fmt.Errorf("No results found on benchmarker execution")
	}

	return &api.Benchmarks{
		Version:           ctrls[0].Version,
		KubernetesVersion: ctrls[0].Version,
		Type:              api.TypeKubernetes,
		NodeName:          nodename,
		Timestamp:         metav1.Time{Time: ts},
		Tests:             benchmark.TestsFromKubeBenchControls(ctrls),
	}, nil
}
