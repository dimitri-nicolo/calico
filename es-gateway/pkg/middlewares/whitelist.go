// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package middlewares

import (
	"net/http"
	"regexp"
	"strings"
)

// TODO: Alina - switch to calico cloud indices
var fieldCapsRegexp = regexp.MustCompile("(/tigera_secure_ee_)(.+)(\\*)(/_field_caps)")
var asyncSearchRegexp = regexp.MustCompile("(/tigera_secure_ee_)(.+)(\\*)(/_async_search)")

func IsWhiteListed(r *http.Request) (allow bool, inspectBody bool) {
	switch {
	// All requests that are whitelisted below are needed to mark Kibana up and running
	case r.URL.Path == "/_bulk" && r.Method == http.MethodPost:
		// This is a request Kibana makes to update its indices
		// POST /_bulk?refresh=false&_source_includes=originId&require_alias=true
		// {"update":{"_id":"task:endpoint:user-artifact-packager:1.0.0","_index":".kibana_task_manager_7.17.18"}}
		// We need to filter through the body of this request and determine if we access any other index that .kibana*
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/docs-bulk.html
		return true, true
	case strings.HasPrefix(r.URL.Path, "/.kibana"):
		// These request access kibana indices for read/write/update data
		// DELETE /.kibana_task_manager_7.17.18/_doc/
		// GET /.kibana_7.17.18/_doc/
		// GET /.kibana%2C.kibana_7.17.18?ignore_unavailable=true
		// GET /.kibana_task_manager%2C.kibana_task_manager_7.17.18?ignore_unavailable=true
		// GET /.kibana-event-log-*/_alias
		// GET /.kibana-event-log-*/_settings
		// GET /.kibana_security_session_1/_doc
		// GET /.kibana_task_manager_7.17.18/_doc/
		// POST /.kibana_7.17.18_001/_pit?keep_alive=10m
		// POST /.kibana_7.17.18_001/_update_by_query
		// POST /.kibana_7.17.18/_search
		// POST /.kibana_7.17.18/_update
		// POST /.kibana_task_manager_7.17.18_001/_pit?keep_alive=10m
		// POST /.kibana_task_manager_7.17.18_001/_update_by_query
		// POST /.kibana_task_manager/_search
		// POST /.kibana_task_manager/_update_by_query
		// PUT /.kibana_7.17.18_001/_mapping?timeout=60s
		// PUT /.kibana_7.17.18/_create
		// PUT /.kibana_7.17.18/_doc
		// PUT /.kibana_task_manager_7.17.18_001/_mapping?
		// PUT /.kibana_task_manager_7.17.18/_create
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/rest-apis.html
		return true, false
	case r.URL.Path == "/_nodes" && r.Method == http.MethodGet:
		// This is a period request Kibana makes to gather information about Elastic nodes
		// The following information is retrieved: nodes.*.version,nodes.*.http.publish_address,nodes.*.ip
		// GET /_nodes?filter_path=nodes.*.version%2Cnodes.*.http.publish_address%2Cnodes.*.ip
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/cluster.html#cluster-nodes
		return true, false
	case r.URL.Path == "/_pit" && r.Method == http.MethodDelete:
		// This is a request to delete a point in time. We will allow it without checking the index
		// PIT request are previously make for kibana indices, like the ones below
		// POST /.kibana_task_manager_7.17.18_001/_pit?keep_alive=10m
		// DELETE /_pit
		// {"id":"u961AwETLmtpYmFuYV83LjE3LjE4XzAwMRZ4WmR3Y1FZY1JBYTQwbWVDam5zeGh3ABY0a1RZdEdHMFRIV0hJYXNIUDZTdFVBAAAAAAAAANE4FnZXUFZrMjdMVENlTFFqSUhxS3VFX1EAARZ4WmR3Y1FZY1JBYTQwbWVDam5zeGh3AAA="}
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/point-in-time-api.html
		return true, true
	case strings.HasPrefix(r.URL.Path, "/_tasks/") && r.Method == http.MethodGet:
		// This is a request from Kibana to access task APIs
		// This request is needed for Kibana to be marked Running
		// GET /_tasks/4kTYtGG0THWHIasHP6StUA%3A658066?wait_for_completion=true&timeout=60s
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/current/tasks.html#tasks-api-path-params
		return true, false
	case r.URL.Path == "/_template/.kibana" && r.Method == http.MethodHead:
		// This request checks the existence of template ./_template/.kibana
		// HEAD /_template/.kibana
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-template-exists-v1.html
		return true, false
	case r.URL.Path == "/_template/kibana_index_template*" && r.Method == http.MethodGet:
		// This request retrieves all index templates that start with kibana_index_template
		// GET /_template/kibana_index_template*
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-get-template.html
		return true, false
	case r.URL.Path == "/_search" && r.Method == http.MethodPost:
		// This is a search request that does not specify the index in the path. This needs special handling to determine
		// if we support the query or not. For example, search requests with a point in time do not
		// specify the index in the URL. Kibana makes _search requests with a point in time during startup.
		// This request is needed for Kibana to be marked Running
		// POST /_search?allow_partial_search_results=false
		// {
		//  "sort": {
		//    "_shard_doc": {
		//      "order": "asc"
		//    }
		//  },
		//  "pit": {
		//    "id": "u961AwETLmtpYmFuYV83LjE3LjE4XzAwMRZ4WmR3Y1FZY1JBYTQwbWVDam5zeGh3ABY0a1RZdEdHMFRIV0hJYXNIUDZTdFVBAAAAAAAAAD_TFnZXUFZrMjdMVENlTFFqSUhxS3VFX1EAARZ4WmR3Y1FZY1JBYTQwbWVDam5zeGh3AAA=",
		//    "keep_alive": "10m"
		//  }..}
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/point-in-time-api.html
		return true, true
	case r.URL.Path == "/_security/privilege/kibana-.kibana" && r.Method == http.MethodGet:
		// This request retrieves privileges for application kibana-.kibana
		// GET /_security/privilege/kibana-.kibana
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/security-api-get-privileges.html
		return true, false
	case r.URL.Path == "/_security/user/_has_privileges" && r.Method == http.MethodPost:
		// This requests checks what privileges has application kibana-.kibana
		// POST /_security/user/_has_privileges
		// {"index":[],"application":[{"application":"kibana-.kibana","resources":["*"],"privileges":["version:7.17.18","login:","ui:7.17.18:enterpriseSearch/all"]}]
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/security-api-has-privileges.html
		return true, false
	case r.URL.Path == "/_xpack" && r.Method == http.MethodGet:
		// This is a request Kibana makes under format
		// GET /_xpack?accept_enterprise=true
		// This request retrieves license details
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/info-api.html
		return true, false

	// All requests that are whitelisted below are needed to load Discovery and Dashboards
	case asyncSearchRegexp.MatchString(r.URL.Path) && r.Method == http.MethodPost:
		// This is a request Kibana makes when loading Discovery and Dashboards
		// This will start an async search request. We expect to have the query
		// defined inside the body at this step. We will allow async requests
		// only for calico indices and enhance them with tenancy enforcement
		// POST /tigera_secure_ee_flows*/_async_search
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/async-search.html
		return true, true
	case strings.HasPrefix(r.URL.Path, "/_async_search") && r.Method == http.MethodGet:
		// This is a request Kibana makes when loading Discovery and Dashboards
		// This will retrieve partial results from the previous issued query
		// We will restrict creation of async searches requests to calico indices and
		// enhance them with a tenancy enforcement. Thus, these requests will be allowed
		// GET /_async_search/FnF4REF0THh5U2gtM3Q0eVpMdWltSmcdNGtUWXRHRzBUSFdISWFzSFA2U3RVQToxMTUwMTY=
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/async-search.html
		return true, false
	case strings.HasPrefix(r.URL.Path, "/_async_search") && r.Method == http.MethodDelete:
		// This is a request Kibana makes when loading Discovery and Dashboards
		// This will delete a previously started async search requests
		// We will restrict creation of async searches requests to calico indices and
		// enhance them with a tenancy enforcement. Thus, these requests will be allowed
		// DELETE /_async_search/FnF4REF0THh5U2gtM3Q0eVpMdWltSmcdNGtUWXRHRzBUSFdISWFzSFA2U3RVQToxMTUwMTY=
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/async-search.html
		return true, false
	case fieldCapsRegexp.MatchString(r.URL.Path) && r.Method == http.MethodGet:
		// This is a request Kibana makes when loading Discovery and Dashboards
		// We will limit this API only for calico indices
		// GET /tigera_secure_ee_flows*/_field_caps
		// Elastic API: https: //www.elastic.co/guide/en/elasticsearch/reference/7.17/search-field-caps.html
		return true, false
	case r.URL.Path == "/_mget" && r.Method == http.MethodPost:
		// This is a request Kibana makes when loading Discovery and Dashboards
		// POST /_mget
		// {"docs":[{"_id":"dashboard:3a849d80-e970-11ea-83c8-edded0d3c4d6","_index":".kibana_7.17.18"}]}
		// We need to filter through the body of this request and determine if we
		// access any other index that .kibana*
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/docs-multi-get.html
		return true, true
	case r.URL.Path == "/_security/_authenticate" && r.Method == http.MethodGet:
		// This request is used for users to log in
		// GET /_security/_authenticate
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/security-api-authenticate.html
		return true, false

	// All requests are needed by event log plugin
	// https://github.com/elastic/kibana/blob/8.13/x-pack/plugins/event_log/README.md
	case strings.HasPrefix(r.URL.Path, "/_alias/.kibana-event-log") && r.Method == http.MethodHead:
		// This request is needed by the event log plugin
		// HEAD /_alias/.kibana-event-log-7.17.18
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-get-alias.html
		return true, false
	case r.URL.Path == "/_ilm/policy/kibana-event-log-policy" && r.Method == http.MethodGet:
		// This request is needed by the event log plugin
		// GET /_ilm/policy/kibana-event-log-policy
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/ilm-get-lifecycle.html
		return true, false
	case strings.HasPrefix(r.URL.Path, "/_index_template/.kibana-event-log") && r.Method == http.MethodHead:
		// This request is needed by the event log plugin
		// HEAD /_index_template/.kibana-event-log-7.17.18-template
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-template-exists-v1.html
		return true, false
	case strings.HasPrefix(r.URL.Path, "/_template/.kibana-event-log") && r.Method == http.MethodHead:
		// This request is needed by the event log plugin
		// HEAD /_template/.kibana-event-log-7.17.18-template
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-template-exists-v1.html
		return true, false
	case r.URL.Path == "/_template/.kibana-event-log-*" && r.Method == http.MethodGet:
		// This request is needed by the event log plugin
		// GET /_template/.kibana-event-log-*
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-get-template.html
		return true, false

	// All requests below are needed by apm plugin
	// https://github.com/elastic/kibana/tree/8.13/x-pack/plugins/apm
	case r.URL.Path == "/.apm-agent-configuration" && r.Method == http.MethodHead:
		// This request is needed by the apm plugin
		// HEAD /.apm-agent-configuration
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-template-exists-v1.html
		return true, false
	case r.URL.Path == "/.apm-agent-configuration/_mapping" && r.Method == http.MethodPut:
		// This request is needed by the apm plugin
		// PUT /.apm-agent-configuration/_mapping
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-put-mapping.html
		return true, false
	case r.URL.Path == "/.apm-custom-link" && r.Method == http.MethodHead:
		// This request is needed by the apm plugin
		// HEAD /.apm-custom-link
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-template-exists-v1.html
		return true, false
	case r.URL.Path == "/.apm-custom-link/_mapping" && r.Method == http.MethodPut:
		// This request is needed by the apm plugin
		// PUT /.apm-custom-link/_mapping
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-put-mapping.html

	// All requests are needed by the monitoring plugin
	// https://github.com/elastic/kibana/tree/8.13/x-pack/plugins/monitoring
	case r.URL.Path == "/_monitoring/bulk" && r.Method == http.MethodPost:
		// This request is needed by the monitor plugin
		// POST /_monitoring/bulk?system_id=kibana&system_api_version=7&interval=10000ms
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/docs-bulk.html
		return true, false

	// All requests are needed by the reporting plugin
	// https://github.com/elastic/kibana/tree/8.13/x-pack/plugins/reporting
	case r.URL.Path == "/.reporting-*/_search" && r.Method == http.MethodPost:
		// This request is needed by the reporting plugin
		// POST /.reporting-*/_search?size=1&seq_no_primary_term=true&_source_excludes=output
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/search-search.html
		return true, false
	case r.URL.Path == "/_ilm/policy/kibana-reporting" && r.Method == http.MethodGet:
		// This request is needed by the reporting plugin
		// GET /_ilm/policy/kibana-reporting
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/ilm-get-lifecycle.html
		return true, false

	// All requests below are needed by ruleRegistry plugin
	// https://github.com/elastic/kibana/blob/main/x-pack/plugins/rule_registry/README.md
	case r.URL.Path == "/_component_template/.alerts-ecs-mappings" && r.Method == http.MethodGet:
		// This request is needed by the ruleRegistry plugin
		// GET /_component_template/.alerts-ecs-mappings
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/getting-component-templates.html
		return true, false
	case r.URL.Path == "/_component_template/.alerts-ecs-mappings" && r.Method == http.MethodPut:
		// This request is needed by the ruleRegistry plugin
		// PUT /_component_template/.alerts-ecs-mappings
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-component-template.html
		return true, false
	case r.URL.Path == "/_component_template/.alerts-observability.apm.alerts-mappings" && r.Method == http.MethodPut:
		// This request is needed by the ruleRegistry plugin
		// PUT /_component_template/.alerts-observability.apm.alerts-mappings
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-component-template.html
		return true, false
	case r.URL.Path == "/_component_template/.alerts-observability.logs.alerts-mappings" && r.Method == http.MethodPut:
		// This request is needed by the ruleRegistry plugin
		// PUT /_component_template/.alerts-observability.logs.alerts-mappings
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-component-template.html
		return true, false
	case r.URL.Path == "/_component_template/.alerts-observability.metrics.alerts-mappings" && r.Method == http.MethodPut:
		// This request is needed by the ruleRegistry plugin
		// PUT /_component_template/.alerts-observability.metrics.alerts-mappings
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-component-template.html
		return true, false
	case r.URL.Path == "/_component_template/.alerts-observability.uptime.alerts-mappings" && r.Method == http.MethodPut:
		// This request is needed by the ruleRegistry plugin
		// PUT /_component_template/.alerts-observability.uptime.alerts-mappings
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-component-template.html
		return true, false
	case r.URL.Path == "/_component_template/.alerts-technical-mappings" && r.Method == http.MethodPut:
		// This request is needed by the ruleRegistry plugin
		// PUT /_component_template/.alerts-technical-mappings
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-component-template.html
		return true, false
	case r.URL.Path == "/_ilm/policy/.alerts-ilm-policy" && r.Method == http.MethodPut:
		// This request is needed by the ruleRegistry plugin
		// GET /_ilm/policy/.alerts-ilm-policy
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/ilm-get-lifecycle.html
		return true, false

	// ALl requests below are needed by the security plugin
	// https://github.com/elastic/kibana/tree/8.13/x-pack/plugins/security
	case r.URL.Path == "/_index_template/.kibana_security_session_index_template_1" && r.Method == http.MethodHead:
		// This request is needed by the security plugin
		// HEAD /_index_template/.kibana_security_session_index_template_1
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-template-exists-v1.html
		return true, false
	case r.URL.Path == "/_template/.kibana_security_session_index_template_1" && r.Method == http.MethodHead:
		// This request checks the existence of template .kibana_security_session_index_template_1
		// This request is needed by the security plugin
		// HEAD /_template/.kibana_security_session_index_template_1
		// Elastic API: https://www.elastic.co/guide/en/elasticsearch/reference/7.17/indices-template-exists-v1.html
		return true, false

	default:
		return false, false
	}

	return false, false
}
