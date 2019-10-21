// Copyright (c) 2019 Tigera Inc. All rights reserved.

package job

import (
	"context"
	"testing"

	"github.com/tigera/intrusion-detection/controller/pkg/util"

	. "github.com/onsi/gomega"
	libcalicov3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/intrusion-detection/controller/pkg/alert/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/alert/statser"
)

func TestJob(t *testing.T) {
	g := NewWithT(t)

	alert := &v3.GlobalAlert{
		ObjectMeta: v1.ObjectMeta{
			Name: "test",
		},
		Spec: libcalicov3.GlobalAlertSpec{
			Description: "test",
			Severity:    100,
			DataSet:     "dns",
		},
	}
	st := &statser.MockStatser{}
	c := elastic.NewMockAlertsController()
	j := NewJob(alert, st, c).(*job)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	j.Run(ctx)

	alert2 := alert.DeepCopy()
	alert2.Spec.Description = "test2"

	g.Eventually(func() *v3.GlobalAlert { return j.alert }).Should(Equal(alert))
	g.Eventually(func() interface{} { return c.Bodies() }).Should(HaveKey(alert.Name))
	g.Eventually(
		func() string { return c.Bodies()[alert.Name].Actions[elastic.IndexActionName].Transform.Source },
	).Should(ContainSubstring(util.PainlessFmt(elastic.GenerateDescriptionFunction(alert.Spec.Description))))

	j.SetAlert(alert2)
	g.Eventually(func() *v3.GlobalAlert { return j.alert }).Should(Equal(alert2))
	g.Eventually(
		func() string { return c.Bodies()[alert.Name].Actions[elastic.IndexActionName].Transform.Source },
	).Should(ContainSubstring(util.PainlessFmt(elastic.GenerateDescriptionFunction(alert2.Spec.Description))),
		"Validate that the generated code has been updated")
}
