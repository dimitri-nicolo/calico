// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.

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
	testClusterID = "badbeefcovfefe"
	updateFreq    = time.Duration(1 * time.Second)

	localstackEndpoint = "http://localhost:4572"
	fakeAWSregion      = "us-south-side-5"
	fakeAWSID          = "much-fake"
	fakeAWSsecrete     = "wow-such-secrete"

	ingressRule1Deny = &calc.RuleID{
		Action:   rules.RuleActionDeny,
		Index:    0,
		IndexStr: "0",
		PolicyID: calc.PolicyID{
			Name:      "popopolicy",
			Namespace: "",
			Tier:      "default",
		},
		Direction: rules.RuleDirIngress,
	}

	egressRule1Deny = &calc.RuleID{
		Action:   rules.RuleActionDeny,
		Index:    1,
		IndexStr: "1",
		PolicyID: calc.PolicyID{
			Name:      "cool-policy",
			Namespace: "",
			Tier:      "cry-me-a-tier",
		},
		Direction: rules.RuleDirEgress,
	}

	muDenyIngress = MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tuple1,
		ruleIDs:      []*calc.RuleID{ingressRule1Deny},
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 5,
			deltaBytes:   25,
		},
	}

	muDenyEgress = MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tuple2,
		ruleIDs:      []*calc.RuleID{egressRule1Deny},
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 10,
			deltaBytes:   55,
		},
	}

	muDenyExpireEgress = MetricUpdate{
		updateType:   UpdateTypeExpire,
		tuple:        tuple1,
		ruleIDs:      []*calc.RuleID{egressRule1Deny},
		isConnection: true,
		inMetric: MetricValue{
			deltaPackets: 7,
			deltaBytes:   69,
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
		cwAPI = testutil.NewMockCloudWatchMetricsClient(dpMetricName, cwCustomNamespace, updateFreq)
		dis = NewCloudWatchMetricsDispatcher(cwAPI)
		agg = NewCloudWatchMetricsAggregator(testClusterID)
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

		go rep.run()
	})
	It("reports one denied packet metric update with 5 denied packet count", func() {
		rep.Report(muDenyIngress)
		time.Sleep(3 * time.Second)

		metrics, err := cwAPI.ListMetrics(&cloudwatch.ListMetricsInput{
			Namespace: aws.String(cwCustomNamespace),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(len(metrics.Metrics)).Should(Equal(1))
	})
})
