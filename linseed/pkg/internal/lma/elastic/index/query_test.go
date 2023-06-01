// Copyright 2021 Tigera Inc. All rights reserved.
package index_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/projectcalico/calico/linseed/pkg/internal/lma/elastic/index"
)

var _ = Describe("Query Converter", func() {
	Context("Alerts", func() {
		It("should return an error if the key is invalid", func() {
			query := "invalid_key=allow"
			_, err := Alerts().NewSelectorQuery(query)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("Invalid selector (invalid_key=allow) in request: invalid key: invalid_key"))
		})

		It("should handle a simple clause", func() {
			result := JsonObject{
				"term": JsonObject{
					"alert": JsonObject{
						"value": "aval1",
					},
				},
			}
			query := "alert=aval1"
			esquery, err := Alerts().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})

		It("should handle an AND clause", func() {
			result := JsonObject{
				"bool": JsonObject{
					"must": []JsonObject{
						{
							"term": JsonObject{
								"alert": JsonObject{
									"value": "aval1",
								},
							},
						},
						{
							"term": JsonObject{
								"type": JsonObject{
									"value": "global_alert",
								},
							},
						},
					},
				},
			}

			query := "alert=aval1 AND type=global_alert"
			esquery, err := Alerts().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})

		It("should handle an OR clause", func() {
			result := JsonObject{
				"bool": JsonObject{
					"should": []JsonObject{
						{
							"term": JsonObject{
								"alert": JsonObject{
									"value": "aval1",
								},
							},
						},
						{
							"term": JsonObject{
								"type": JsonObject{
									"value": "global_alert",
								},
							},
						},
					},
				},
			}
			query := "alert=aval1 OR type=global_alert"
			esquery, err := Alerts().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})

		It("should handle an composite clause", func() {
			result := JsonObject{
				"bool": JsonObject{
					"must": []JsonObject{
						{
							"bool": JsonObject{
								"should": []JsonObject{
									{
										"term": JsonObject{
											"alert": JsonObject{
												"value": "aval1",
											},
										},
									},
									{
										"term": JsonObject{
											"type": JsonObject{
												"value": "global_alert",
											},
										},
									},
								},
							},
						},
						{
							"term": JsonObject{
								"_id": JsonObject{
									"value": "idval",
								},
							},
						},
					},
				},
			}
			query := "(alert=aval1 OR type=global_alert) AND _id=idval"
			esquery, err := Alerts().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})

		It("should handle a clause to filter DNS queries", func() {
			// This test simulate a filter query done on the UI to filter alerts related to DNS query.
			// Currently these are implemented using the "filter" param handled by es-proxy for /events/search
			// and we want to let linseed handle them with a POST /events (List) query using a selector
			// to capture the same logic.
			//
			// Existing "filter" param sample value:
			// [
			// 	{
			// 		"range":{
			// 			"time":{
			// 				"gte":"2023-05-31T00:00:00Z",
			// 				"lte":"2023-06-02T23:59:59Z"
			// 			}
			// 		}
			// 	},
			// 	{
			// 		"terms":{
			// 			"type":[
			// 				"suspicious_dns_query",
			// 				"gtf_suspicious_dns_query"
			// 			]
			// 		}
			// 	},
			// 	{
			// 		"wildcard":{
			// 			"source_name":"*basic-123*"
			// 		}
			// 	},
			// 	{
			// 		"wildcard":{
			// 			"source_namespace":"*default*"
			// 		}
			// 	},
			// 	{
			// 		"range":{
			// 			"source_ip":{
			// 				"gte":"123.4.5.6",
			// 				"lte":"123.4.5.9"
			// 			}
			// 		}
			// 	},
			// 	{
			// 		"terms":{
			// 			"suspicious_domains":[
			// 				"sysdig.com", "cilium.com"
			// 			]
			// 		}
			// 	}
			// ]

			// A few observations to keep the same logic using a selector:
			//
			// 1. The filter for time used in the UI
			// {
			// 		"range":{
			// 			"time":{
			// 				"gte":"2023-05-31T00:00:00Z",
			// 				"lte":"2023-06-02T23:59:59Z"
			// 			}
			// 		}
			// 	},
			//   can be achieved with a selector "'time' >= '2023-05-31T00:00:00Z' AND time <= '2023-06-02T23:59:59Z'"
			//   that generates the following terms
			// {
			// 		"range":{
			// 			"time":{
			// 				"gte":"2023-05-31T00:00:00Z",
			// 			}
			// 		}
			// 	},
			// {
			// 		"range":{
			// 			"time":{
			// 				"lte":"2023-06-02T23:59:59Z"
			// 			}
			// 		}
			// 	},
			//
			// 2. The filter for "source_namespace" use a wildcard query and adds * before and  after a user input.
			//    For example, user input "default" will generate the following filter:
			// {
			// 		"wildcard":{
			// 			"source_namespace":"*default*"
			// 		}
			// 	},
			//    Selector "source_namespace = default" generates:
			//  {
			//      "term":{
			// 	        "source_name":{
			// 		        "value": "basic-123"
			// 	        }
			//      }
			//   },
			//    which will achieve the same result in most cases.
			//    However it will not work as expected when the user inputs a * like in "my-namespace-*-dev".
			//    We probably don't need to care. If we do, we could update converter.go to output a
			//    "wildcard" term when the value contains a *...
			//
			// 3. Filter with multiple values use a terms query what we don't support.
			//    Sample filter with multiple values generated by the UI:
			// 	{
			// 		"terms":{
			// 			"suspicious_domains":[
			// 				"sysdig.com", "cilium.com"
			// 			]
			// 		}
			// 	}
			//    This can be emulated with a selector like "suspicious_domains in {'sysdig.com','cilium.io'}"
			//    that generates the following filters:
			// {
			// 	"bool": {
			// 		"should": []{
			// 			{
			// 				"wildcard": {
			// 					"suspicious_domains": {
			// 						"value": "sysdig.com"
			// 					}
			// 				}
			// 			},
			// 			{
			// 				"wildcard": {
			// 					"suspicious_domains": {
			// 						"value": "cilium.io"
			// 					}
			// 				}
			// 			}
			// 		}
			// 	}
			// }

			result := JsonObject{
				"bool": JsonObject{
					"must": []JsonObject{
						{
							"range": JsonObject{
								"time": JsonObject{
									"gte": "2023-05-31T00:00:00Z",
								},
							},
						},
						{
							"range": JsonObject{
								"time": JsonObject{
									"lte": "2023-06-02T23:59:59Z",
								},
							},
						},
						{
							"bool": JsonObject{
								"should": []JsonObject{
									{
										"wildcard": JsonObject{
											"type": JsonObject{
												"value": "suspicious_dns_query",
											},
										},
									},
									{
										"wildcard": JsonObject{
											"type": JsonObject{
												"value": "gtf_suspicious_dns_query",
											},
										},
									},
								},
							},
						},
						{
							"term": JsonObject{
								"source_name": JsonObject{
									"value": "basic-123",
								},
							},
						},
						{
							"term": JsonObject{
								"source_namespace": JsonObject{
									"value": "default",
								},
							},
						},
						{
							"bool": JsonObject{
								"should": []JsonObject{
									{
										"wildcard": JsonObject{
											"suspicious_domains": JsonObject{
												"value": "sysdig.com",
											},
										},
									},
									{
										"wildcard": JsonObject{
											"suspicious_domains": JsonObject{
												"value": "cilium.io",
											},
										},
									},
								},
							},
						},
					},
				},
			}

			query := "'time' >= '2023-05-31T00:00:00Z' AND time <= '2023-06-02T23:59:59Z' and " +
				"type IN { suspicious_dns_query, gtf_suspicious_dns_query} AND " +
				"\"source_name\"=\"basic-123\" AND \"source_namespace\" = \"default\" AND " +
				"suspicious_domains in {'sysdig.com','cilium.io'}"
			esquery, err := Alerts().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})
	})

	Context("Dns", func() {
		It("should return an error if the key is invalid", func() {
			query := "invalid_key=allow"
			_, err := DnsLogs().NewSelectorQuery(query)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("Invalid selector (invalid_key=allow) in request: " +
				"invalid key: invalid_key"))
		})

		It("should return an error if the value is invalid", func() {
			query := "client_ip=invalid_value"
			_, err := DnsLogs().NewSelectorQuery(query)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("Invalid selector (client_ip=invalid_value) in request: " +
				"invalid value for client_ip: invalid_value"))
		})

		It("should handle a simple clause", func() {
			result := JsonObject{
				"term": JsonObject{
					"client_ip": JsonObject{
						"value": "1.0.1.5",
					},
				},
			}
			query := "client_ip=\"1.0.1.5\""
			esquery, err := DnsLogs().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})

		It("should handle an AND clause", func() {
			result := JsonObject{
				"bool": JsonObject{
					"must": []JsonObject{
						{
							"term": JsonObject{
								"start_time": JsonObject{
									"value": "2006-01-02 15:04:05",
								},
							},
						},
						{
							"term": JsonObject{
								"client_ip": JsonObject{
									"value": "10.0.0.1",
								},
							},
						},
					},
				},
			}

			query := "start_time=\"2006-01-02 15:04:05\" AND client_ip=\"10.0.0.1\""
			esquery, err := DnsLogs().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})

		It("should handle an OR clause", func() {
			result := JsonObject{
				"bool": JsonObject{
					"should": []JsonObject{
						{
							"term": JsonObject{
								"qname": JsonObject{
									"value": "http://www.yolo.com",
								},
							},
						},
						{
							"term": JsonObject{
								"count": JsonObject{
									"value": "5",
								},
							},
						},
					},
				},
			}
			query := "qname=\"http://www.yolo.com\" OR count=5"
			esquery, err := DnsLogs().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})

		It("should handle an composite clause", func() {
			result := JsonObject{
				"bool": JsonObject{
					"must": []JsonObject{
						{
							"bool": JsonObject{
								"should": []JsonObject{
									{
										"term": JsonObject{
											"end_time": JsonObject{
												"value": "2006-01-02 15:04:05",
											},
										},
									},
									{
										"term": JsonObject{
											"count": JsonObject{
												"value": "225",
											},
										},
									},
								},
							},
						},
						{
							"term": JsonObject{
								"client_ip": JsonObject{
									"value": "192.168.2.1",
								},
							},
						},
					},
				},
			}
			query := "(end_time=\"2006-01-02 15:04:05\" OR count=225) AND client_ip=\"192.168.2.1\""
			esquery, err := DnsLogs().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})
	})

	Context("Flow", func() {
		It("should return an error if the key is invalid", func() {
			query := "invalid_key=allow"
			_, err := FlowLogs().NewSelectorQuery(query)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("Invalid selector (invalid_key=allow) in request: " +
				"invalid key: invalid_key"))
		})

		It("should handle a simple clause", func() {
			result := JsonObject{
				"term": JsonObject{
					"action": JsonObject{
						"value": "allow",
					},
				},
			}
			query := "action=allow"
			esquery, err := FlowLogs().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})

		It("should handle an AND clause", func() {
			result := JsonObject{
				"bool": JsonObject{
					"must": []JsonObject{
						{
							"term": JsonObject{
								"action": JsonObject{
									"value": "allow",
								},
							},
						},
						{
							"term": JsonObject{
								"action": JsonObject{
									"value": "deny",
								},
							},
						},
					},
				},
			}

			query := "action=allow AND action=deny"
			esquery, err := FlowLogs().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})

		It("should handle an OR clause", func() {
			result := JsonObject{
				"bool": JsonObject{
					"should": []JsonObject{
						{
							"term": JsonObject{
								"action": JsonObject{
									"value": "allow",
								},
							},
						},
						{
							"term": JsonObject{
								"action": JsonObject{
									"value": "deny",
								},
							},
						},
					},
				},
			}
			query := "action=allow OR action=deny"
			esquery, err := FlowLogs().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})

		It("should handle an composite clause", func() {
			result := JsonObject{
				"bool": JsonObject{
					"must": []JsonObject{
						{
							"bool": JsonObject{
								"should": []JsonObject{
									{
										"term": JsonObject{
											"action": JsonObject{
												"value": "allow",
											},
										},
									},
									{
										"term": JsonObject{
											"action": JsonObject{
												"value": "deny",
											},
										},
									},
								},
							},
						},
						{
							"term": JsonObject{
								"action": JsonObject{
									"value": "deny",
								},
							},
						},
					},
				},
			}
			query := "(action=allow OR action=deny) AND action=deny"
			esquery, err := FlowLogs().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})
	})

	Context("L7", func() {
		It("should return an error if the key is invalid", func() {
			query := "invalid_key=allow"
			_, err := L7Logs().NewSelectorQuery(query)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("Invalid selector (invalid_key=allow) in request: " +
				"invalid key: invalid_key"))
		})

		It("should return an error if the value is invalid", func() {
			query := "source_type=invalid_value"
			_, err := L7Logs().NewSelectorQuery(query)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("Invalid selector (source_type=invalid_value) in request: " +
				"invalid value for source_type: invalid_value"))
		})

		It("should handle a simple clause", func() {
			result := JsonObject{
				"term": JsonObject{
					"source_type": JsonObject{
						"value": "wep",
					},
				},
			}
			query := "source_type=wep"
			esquery, err := L7Logs().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})

		It("should handle an AND clause", func() {
			result := JsonObject{
				"bool": JsonObject{
					"must": []JsonObject{
						{
							"term": JsonObject{
								"duration_mean": JsonObject{
									"value": "50",
								},
							},
						},
						{
							"term": JsonObject{
								"dest_type": JsonObject{
									"value": "net",
								},
							},
						},
					},
				},
			}

			query := "duration_mean=50 AND dest_type=net"
			esquery, err := L7Logs().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})

		It("should handle an OR clause", func() {
			result := JsonObject{
				"bool": JsonObject{
					"should": []JsonObject{
						{
							"term": JsonObject{
								"url": JsonObject{
									"value": "http://www.yolo.com",
								},
							},
						},
						{
							"term": JsonObject{
								"dest_service_port_num": JsonObject{
									"value": "65535",
								},
							},
						},
					},
				},
			}
			query := "url=\"http://www.yolo.com\" OR dest_service_port_num=65535"
			esquery, err := L7Logs().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})

		It("should handle an composite clause", func() {
			result := JsonObject{
				"bool": JsonObject{
					"must": []JsonObject{
						{
							"bool": JsonObject{
								"should": []JsonObject{
									{
										"term": JsonObject{
											"url": JsonObject{
												"value": "http://www.yolo.com",
											},
										},
									},
									{
										"term": JsonObject{
											"method": JsonObject{
												"value": "methodval",
											},
										},
									},
								},
							},
						},
						{
							"term": JsonObject{
								"dest_type": JsonObject{
									"value": "ns",
								},
							},
						},
					},
				},
			}
			query := "(url=\"http://www.yolo.com\" OR method=methodval) AND dest_type=ns"
			esquery, err := L7Logs().NewSelectorQuery(query)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(esquery.Source()).Should(BeEquivalentTo(result))
		})
	})
})
