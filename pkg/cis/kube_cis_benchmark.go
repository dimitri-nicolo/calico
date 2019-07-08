package cis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/aquasecurity/kube-bench/check"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/compliance/pkg/benchmark"
)

// Benchmarker implements benchmark.Executor
type Benchmarker struct {
}

// NewBenchmarker returns a benchmark.Executor instance that can execute kubernetes cis benchmark tests
func NewBenchmarker() benchmark.BenchmarksExecutor {
	return new(Benchmarker)
}

// ExecuteBenchmarks determines the appropriate benchmarker to run for the given benchmark type.
func (b *Benchmarker) ExecuteBenchmarks(ctx context.Context, ct benchmark.BenchmarkType, nodename string) (*benchmark.Benchmarks, error) {
	if ct == benchmark.TypeKubernetes {
		return b.executeKubeBenchmark(ctx, nodename)
	}
	return nil, fmt.Errorf("No handler found for benchmark type %s", ct)
}

// executeKubeBenchmark executes kube-bench.
func (b *Benchmarker) executeKubeBenchmark(ctx context.Context, nodename string) (*benchmark.Benchmarks, error) {
	// Determine Openshift args if any.
	args, err := determineOpenshiftArgs(nodename)
	if err != nil {
		return nil, err
	}
	args = append(args, "--json")
	log.WithField("cmd", args).Debug("executing benchmarker")

	// Execute the benchmarker
	ts := time.Now()
	cmd := exec.Command("kube-bench", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.WithError(err).Error("Failed to execute kubernetes benchmarker")
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

	return &benchmark.Benchmarks{
		Version:           ctrls[0].Version,
		KubernetesVersion: ctrls[0].Version,
		Type:              benchmark.TypeKubernetes,
		NodeName:          nodename,
		Timestamp:         metav1.Time{Time: ts},
		Tests:             benchmark.TestsFromKubeBenchControls(ctrls),
	}, nil
}
