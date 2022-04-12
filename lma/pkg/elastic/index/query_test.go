// Copyright 2021 Tigera Inc. All rights reserved.
package index_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/tigera/lma/pkg/elastic/index"
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
