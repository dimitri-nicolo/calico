package scaleloader

const (
	auditV1Template = `
{"kind":"Event","apiVersion":"audit.k8s.io/v1","level":"RequestResponse","auditID":"2a1bf0df-3252-4535-9f4f-6dd97fcad8d1","stage":"ResponseComplete","requestURI":"",
  "verb":"{{.Verb}}",
  "user":{"username":"kubernetes-admin","groups":["system:masters","system:authenticated"]},
  "sourceIPs":["199.116.73.11"],"userAgent":"kubectl/v1.13.4 (darwin/amd64) kubernetes/c27b913",
  "objectRef":{{.ObjectRef}},
  "responseStatus":{"metadata":{},"code":201},
  "requestObject":{},
  "responseObject":{{.ResponseObject}},
  "requestReceivedTimestamp":"{{.Timestamp}}","stageTimestamp":"{{.Timestamp}}",
  "annotations":{"authorization.k8s.io/decision":"allow","authorization.k8s.io/reason":""},
  "name":"default-deny"}
`
	auditV1BetaTemplate = `
{"kind":"Event","apiVersion":"audit.k8s.io/v1beta1",
  "metadata":{"creationTimestamp":"{{.Timestamp}}"},
  "level":"RequestResponse",
  "timestamp":"{{.Timestamp}}",
  "auditID":"a999a013-5e6b-4609-8a1e-d7f215255ed9","stage":"ResponseComplete","requestURI":"/apis/projectcalico.org/v3/globalnetworksets",
  "verb":"{{.Verb}}",
  "user":{"username":"kubernetes-admin","groups":["system:masters","system:authenticated"]},"sourceIPs":["199.116.73.11"],
  "objectRef":{{.ObjectRef}},
  "responseStatus":{"metadata":{},"code":201},
  "requestObject":{},
  "responseObject":{{.ResponseObject}},
  "requestReceivedTimestamp":"{{.Timestamp}}","stageTimestamp":"{{.Timestamp}}",
  "name":"google-dns"}
`
)
