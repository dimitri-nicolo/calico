// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/collector/testutil"
	"github.com/projectcalico/felix/rules"
)

var (
	clusterID  = "badbeefcovfefe"
	updateFreq = time.Duration(5 * time.Second)

	localstackEndpoint = "http://localhost:4572"
	fakeAWSregion      = "us-south-side-5"
	fakeAWSID          = "much-fake"
	fakeAWSsecrete     = "wow-such-secrete"

	ingressRule1Deny = &calc.RuleID{
		Action:    rules.RuleActionDeny,
		Index:     0,
		IndexStr:  "0",
		Name:      "popopolicy",
		Namespace: "",
		Tier:      "default",
		Direction: rules.RuleDirIngress,
	}

	egressRule1Deny = &calc.RuleID{
		Action:    rules.RuleActionDeny,
		Index:     1,
		IndexStr:  "1",
		Name:      "cool-policy",
		Namespace: "",
		Tier:      "cry-me-a-tier",
		Direction: rules.RuleDirEgress,
	}

	muDenyIngress = MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tuple1,
		ruleID:       ingressRule1Deny,
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 5,
			deltaBytes:   25,
		},
	}

	muDenyEgress = MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tuple2,
		ruleID:       egressRule1Deny,
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 10,
			deltaBytes:   55,
		},
	}
)

var _ = Describe("CloudWatch Reporter verification", func() {
	var (
		rep   *cloudWatchMetricReporter
		dis   MetricDispatcher
		agg   MetricAggregator
		cwAPI cloudwatchiface.CloudWatchAPI
	)
	BeforeEach(func() {
		dis = NewCloudWatchMetricsDispatcher(nil)
		agg = NewCloudWatchMetricsAggregator(clusterID)
		rep = newCloudWatchMetricReporter(dis, agg, updateFreq)

		// TODO: use this configuration when localstack is working again
		// localstack metrics are broken: https://github.com/localstack/localstack/issues/606
		//sess := session.Must(session.NewSessionWithOptions(session.Options{
		//	SharedConfigState: session.SharedConfigEnable,
		//}))
		//
		//cwAPI = cloudwatch.New(
		//	sess,
		//	&aws.Config{
		//		Credentials: credentials.NewStaticCredentials(fakeAWSID, fakeAWSsecrete, ""),
		//		Endpoint:    &localstackEndpoint,
		//		Region:      aws.String(fakeAWSregion),
		//		DisableSSL:  aws.Bool(true),
		//	},
		//)

		cwAPI = testutil.NewMockCloudWatchMetricsClient(dpMetricName, cwCustomNamespace)
		go rep.run()
	})
	It("report one denied packet metric update with 5 denied packet count", func() {
		rep.Report(muDenyIngress)
		time.Sleep(7 * time.Second)

		metrics, err := cwAPI.ListMetrics(&cloudwatch.ListMetricsInput{
			Namespace: aws.String(cwCustomNamespace),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(len(metrics.Metrics)).Should(Equal(1))
	})

	It("report multiple denied packet metric updates with different directions", func() {
		rep.Report(muDenyIngress)
		time.Sleep(7 * time.Second)

		rep.Report(muDenyEgress)
		time.Sleep(7 * time.Second)

		metrics, err := cwAPI.ListMetrics(&cloudwatch.ListMetricsInput{
			Namespace: aws.String(cwCustomNamespace),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(len(metrics.Metrics)).Should(Equal(2))
	})
})
