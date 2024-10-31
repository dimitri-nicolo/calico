// Copyright (c) 2019 Tigera, Inc. All rights reserved.
/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"sync"
	"time"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	"github.com/projectcalico/calico/compliance/pkg/api"
	"github.com/projectcalico/calico/compliance/pkg/datastore"
)

const (
	reportPodTemplate       = "tigera.io.report"
	reportPodTemplatePrefix = "tigera.io.report."
	reportContainer         = "reporter"
)

// reportControlInterface is an interface that knows how to update GlobalReport status
// created as an interface to allow testing.
type reportControlInterface interface {
	UpdateStatus(report *v3.GlobalReport) (*v3.GlobalReport, error)
}

// realReportControl is the default implementation of reportControlInterface.
type realReportControl struct {
	clientSet datastore.ClientSet
}

var _ reportControlInterface = &realReportControl{}

func (c *realReportControl) UpdateStatus(report *v3.GlobalReport) (*v3.GlobalReport, error) {
	return c.clientSet.GlobalReports().UpdateStatus(context.Background(), report, metav1.UpdateOptions{})
}

// fakeReportControl is the default implementation of reportControlInterface.
type fakeReportControl struct {
	Updates []v3.GlobalReport
}

var _ reportControlInterface = &fakeReportControl{}

func (c *fakeReportControl) UpdateStatus(report *v3.GlobalReport) (*v3.GlobalReport, error) {
	c.Updates = append(c.Updates, *report)
	return report, nil
}

// ------------------------------------------------------------------ //

// jobControlInterface is an interface that knows how to add or delete jobs
// created as an interface to allow testing.
type jobControlInterface interface {
	// GetJob retrieves a Job.
	GetJob(namespace, name string) (*batchv1.Job, error)
	// CreateJob creates new Jobs according to the spec.
	CreateJob(namespace string, job *batchv1.Job) (*batchv1.Job, error)
	// UpdateJob updates a Job.
	UpdateJob(namespace string, job *batchv1.Job) (*batchv1.Job, error)
	// DeleteJob deletes the Job identified by name.
	// TODO: delete by UID?
	DeleteJob(namespace string, name string) error
}

// realJobControl is the default implementation of jobControlInterface.
type realJobControl struct {
	clientSet clientset.Interface
	Recorder  record.EventRecorder
}

var _ jobControlInterface = &realJobControl{}

func (r *realJobControl) GetJob(namespace, name string) (*batchv1.Job, error) {
	return r.clientSet.BatchV1().Jobs(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (r *realJobControl) UpdateJob(namespace string, job *batchv1.Job) (*batchv1.Job, error) {
	return r.clientSet.BatchV1().Jobs(namespace).Update(context.Background(), job, metav1.UpdateOptions{})
}

func (r *realJobControl) CreateJob(namespace string, job *batchv1.Job) (*batchv1.Job, error) {
	return r.clientSet.BatchV1().Jobs(namespace).Create(context.Background(), job, metav1.CreateOptions{})
}

func (r *realJobControl) DeleteJob(namespace string, name string) error {
	background := metav1.DeletePropagationBackground
	return r.clientSet.BatchV1().Jobs(namespace).Delete(context.Background(), name, metav1.DeleteOptions{PropagationPolicy: &background})
}

type fakeJobControl struct {
	sync.Mutex
	Job           *batchv1.Job
	Jobs          []batchv1.Job
	DeleteJobName []string
	Err           error
	UpdateJobName []string
}

var _ jobControlInterface = &fakeJobControl{}

func (f *fakeJobControl) CreateJob(namespace string, job *batchv1.Job) (*batchv1.Job, error) {
	f.Lock()
	defer f.Unlock()
	if f.Err != nil {
		return nil, f.Err
	}
	f.Jobs = append(f.Jobs, *job)
	job.UID = "test-uid"
	return job, nil
}

func (f *fakeJobControl) GetJob(namespace, name string) (*batchv1.Job, error) {
	f.Lock()
	defer f.Unlock()
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Job, nil
}

func (f *fakeJobControl) UpdateJob(namespace string, job *batchv1.Job) (*batchv1.Job, error) {
	f.Lock()
	defer f.Unlock()
	if f.Err != nil {
		return nil, f.Err
	}
	f.UpdateJobName = append(f.UpdateJobName, job.Name)
	return job, nil
}

func (f *fakeJobControl) DeleteJob(namespace string, name string) error {
	f.Lock()
	defer f.Unlock()
	if f.Err != nil {
		return f.Err
	}
	f.DeleteJobName = append(f.DeleteJobName, name)
	return nil
}

func (f *fakeJobControl) Clear() {
	f.Lock()
	defer f.Unlock()
	f.DeleteJobName = []string{}
	f.Jobs = []batchv1.Job{}
	f.Err = nil
}

// ------------------------------------------------------------------ //

// podControlInterface is an interface that knows how to list or delete pods
// created as an interface to allow testing.
type podControlInterface interface {
	// ListPods list pods
	ListPods(namespace string, opts metav1.ListOptions) (*v1.PodList, error)
	// DeleteJob deletes the pod identified by name.
	// TODO: delete by UID?
	DeletePod(namespace string, name string) error
}

// realPodControl is the default implementation of podControlInterface.
type realPodControl struct {
	clientSet clientset.Interface
	Recorder  record.EventRecorder
}

var _ podControlInterface = &realPodControl{}

func (r realPodControl) ListPods(namespace string, opts metav1.ListOptions) (*v1.PodList, error) {
	return r.clientSet.CoreV1().Pods(namespace).List(context.Background(), opts)
}

func (r realPodControl) DeletePod(namespace string, name string) error {
	return r.clientSet.CoreV1().Pods(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
}

type fakePodControl struct {
	sync.Mutex
	Pods          []v1.Pod
	DeletePodName []string
	Err           error
}

var _ podControlInterface = &fakePodControl{}

func (f *fakePodControl) ListPods(namespace string, opts metav1.ListOptions) (*v1.PodList, error) {
	f.Lock()
	defer f.Unlock()
	if f.Err != nil {
		return nil, f.Err
	}
	return &v1.PodList{Items: f.Pods}, nil
}

func (f *fakePodControl) DeletePod(namespace string, name string) error {
	f.Lock()
	defer f.Unlock()
	if f.Err != nil {
		return f.Err
	}
	f.DeletePodName = append(f.DeletePodName, name)
	return nil
}

// archivedReportQueryInterface is an interface that knows how to query the archived reports to determine the last
// report end timestamp.
type archivedReportQueryInterface interface {
	GetLastReportStartEndTime(name string) (*metav1.Time, *metav1.Time, error)
}

// realArchivedReportQuery is the default implementation of archivedReportQueryInterface.
type realArchivedReportQuery struct {
	reportRetriever api.ReportRetriever
}

var _ archivedReportQueryInterface = &realArchivedReportQuery{}

func (c *realArchivedReportQuery) GetLastReportStartEndTime(name string) (*metav1.Time, *metav1.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	ar, err := c.reportRetriever.RetrieveLastArchivedReportSummary(ctx, name)
	if err != nil {
		return nil, nil, err
	}
	return &ar.StartTime, &ar.EndTime, nil
}

// podTemplateQueryInterface is an interface that knows how to query the archived reports to determine the last
// report end timestamp.
type podTemplateQueryInterface interface {
	GetPodTemplate(namespace, name string) (*v1.PodTemplate, error)
}

// realPodTemplateQuery is the default implementation of podTemplateQueryInterface.
type realPodTemplateQuery struct {
	clientSet datastore.ClientSet
}

var _ podTemplateQueryInterface = &realPodTemplateQuery{}

func (c *realPodTemplateQuery) GetPodTemplate(namespace, name string) (*v1.PodTemplate, error) {
	// Get the rep template. Preferentially look for one called: "tigera.io.rep.<reportname>" and fallback to
	// "tigera.io.rep" if that does not exist.
	podTemplateName := reportPodTemplatePrefix + name
	pt, err := c.clientSet.CoreV1().PodTemplates(namespace).Get(context.Background(), podTemplateName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		podTemplateName = reportPodTemplate
		pt, err = c.clientSet.CoreV1().PodTemplates(namespace).Get(context.Background(), podTemplateName, metav1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}
	return pt, nil
}
