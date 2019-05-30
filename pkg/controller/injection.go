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
	"fmt"
	"sync"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"github.com/tigera/compliance/pkg/datastore"
	"github.com/tigera/compliance/pkg/report"
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
	return c.clientSet.GlobalReports().UpdateStatus(report)
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
	return r.clientSet.BatchV1().Jobs(namespace).Get(name, metav1.GetOptions{})
}

func (r *realJobControl) UpdateJob(namespace string, job *batchv1.Job) (*batchv1.Job, error) {
	return r.clientSet.BatchV1().Jobs(namespace).Update(job)
}

func (r *realJobControl) CreateJob(namespace string, job *batchv1.Job) (*batchv1.Job, error) {
	return r.clientSet.BatchV1().Jobs(namespace).Create(job)
}

func (r *realJobControl) DeleteJob(namespace string, name string) error {
	background := metav1.DeletePropagationBackground
	return r.clientSet.BatchV1().Jobs(namespace).Delete(name, &metav1.DeleteOptions{PropagationPolicy: &background})
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
	job.SelfLink = fmt.Sprintf("/api/batch/v1/namespaces/%s/jobs/%s", namespace, job.Name)
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
	return r.clientSet.CoreV1().Pods(namespace).List(opts)
}

func (r realPodControl) DeletePod(namespace string, name string) error {
	return r.clientSet.CoreV1().Pods(namespace).Delete(name, nil)
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
	reportRetriever report.ReportRetriever
}

var _ archivedReportQueryInterface = &realArchivedReportQuery{}

func (c *realArchivedReportQuery) GetLastReportStartEndTime(name string) (*metav1.Time, *metav1.Time, error) {
	ar, err := c.reportRetriever.RetrieveLastArchivedReportSummary(name)
	if err != nil {
		return nil, nil, err
	}
	return &ar.StartTime, &ar.EndTime, nil
}

// fakeArchivedReportQuery is the default fake implementation of archivedReportQueryInterface.
type fakeArchivedReportQuery struct {
	t   *metav1.Time
	err error
}

var _ reportControlInterface = &fakeReportControl{}

func (c *fakeArchivedReportQuery) GetLastReportStartEndTime(name string) (*metav1.Time, *metav1.Time, error) {
	return c.t, c.t, c.err
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
	pt, err := c.clientSet.CoreV1().PodTemplates(namespace).Get(podTemplateName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		podTemplateName = reportPodTemplate
		pt, err = c.clientSet.CoreV1().PodTemplates(namespace).Get(podTemplateName, metav1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}
	return pt, nil
}

// fakePodTemplateQuery is the default fake implementation of podTemplateQueryInterface.
type fakePodTemplateQuery struct {
	t   *v1.PodTemplate
	err error
}

var _ reportControlInterface = &fakeReportControl{}

func (c *fakePodTemplateQuery) GetPodTemplate(namespace, name string) (*v1.PodTemplate, error) {
	return c.t, c.err
}
