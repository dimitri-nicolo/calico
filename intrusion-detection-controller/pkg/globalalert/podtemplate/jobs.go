package podtemplate

import (
	"time"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CronJobEveryEntryPrefix = "@every"
)

func CreateCronJobFromPodTemplate(name, namepsace string, schedule time.Duration, labels map[string]string, pt v1.PodTemplate) *batchv1.CronJob {

	adjobTemplate := CreateJobFromPodTemplate(name, namepsace, labels, pt)

	cronSchedule := CronJobEveryEntryPrefix + " " + schedule.String()

	adCronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namepsace,
			Labels:    labels,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: cronSchedule,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: adjobTemplate.ObjectMeta,
				Spec:       adjobTemplate.Spec,
			},
		},
	}

	return adCronJob
}

func CreateJobFromPodTemplate(name string, namespace string, labels map[string]string, pt v1.PodTemplate) *batchv1.Job {

	// combine labels from podtemplate
	jobLabels := labels
	for k, v := range pt.Template.Labels {
		jobLabels[k] = v
	}

	template := v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      jobLabels,
			Annotations: pt.Template.Annotations,
		},
		Spec: pt.Template.Spec,
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    jobLabels,
		},
		Spec: batchv1.JobSpec{
			Template: template,
		},
	}

	return job
}
