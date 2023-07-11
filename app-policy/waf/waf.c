// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

#include "waf.h"
#include "_cgo_export.h"
#include "modsecurity/modsecurity.h"
#include "modsecurity/rules_set.h"
#include "modsecurity/transaction.h"
#include "modsecurity/intervention.h"
#include <errno.h>

ModSecurity *modsec = NULL;
RulesSet *rules = NULL;

// Helper return value codes.
typedef enum tag_enum_msc_retval
{
	msc_retval_new_transaction_with_id = 1,
	msc_retval_process_connection,
	msc_retval_process_uri,
	msc_retval_add_request_header,
	msc_retval_process_request_headers,
	msc_retval_append_request_body,
	msc_retval_process_request_body,
	msc_retval_process_logging,
} enum_msc_retval;

// Private helper function to initialize ModSecurity.
static void initializeModSecurityImpl();
//
// Function prototype must match modsecurity.cc ModSecLogCb callback signature.
void CModSecurityLoggingCallback( void *referenceAPI, const void *ruleMessage)
{
	// Remove constness and coerce to char* to be compatible with Golang API
	char *payload = (char *)ruleMessage;
	GoModSecurityLoggingCallback(payload);
}

// General public APIs.
void InitializeModSecurity()
{
	initializeModSecurityImpl();
}

static void initializeModSecurityImpl()
{
	modsec = msc_init();
	msc_set_log_cb(modsec, CModSecurityLoggingCallback);
	rules = msc_create_rules_set();
}

const char* LoadModSecurityCoreRuleSet(char *file)
{
	const char *error = NULL;
	char *error_message = NULL;
	if (modsec == NULL)
	{
		initializeModSecurityImpl();
	}

	msc_rules_add_file(rules, file, &error);
	if (error != NULL)
	{
		error_message = (char *)error;
	}

	error = NULL;
	return error_message;
}

ModSecurityIntervention* NewModSecurityIntervention() {
	ModSecurityIntervention *intervention = malloc(sizeof(
		ModSecurityIntervention));
	return intervention;
}

ModSecurityIntervention *processIntervention(Transaction *transaction)
{
	// Detects Mod Security intervention.
	ModSecurityIntervention in, *ret;
	in.disruptive = 0;
	in.url = NULL;
	in.log = NULL;
	in.status = 200;
	if (msc_intervention(transaction, &in)) {
		ret = malloc(sizeof(ModSecurityIntervention));
		memcpy(ret, &in, sizeof(ModSecurityIntervention));
		return ret;
	}

	return NULL;
}

ModSecurityIntervention *ProcessHttpRequest(int *err, char *id,	char *uri,
	char *http_method, char *http_protocol, char *http_version,
	char *client_host, int client_port, char *server_host, int server_port,
	char **reqHeaderKeys, char **reqHeaderVals, int reqHeaderSize,
	char *reqBodyText, int reqBodySize)
{
	ModSecurityIntervention *retVal = NULL;

	const char *reqHeaderKey = NULL;
	const char *reqHeaderVal = NULL;
	int idx = 0, ret;

	*err = 0;

	if (modsec == NULL)
	{
		initializeModSecurityImpl();
	}

	Transaction *transaction = NULL;
	transaction = msc_new_transaction_with_id(modsec, rules, id, NULL);
	if (transaction == NULL)
	{
		*err = msc_retval_new_transaction_with_id;
		goto out;
	}
	if (retVal = processIntervention(transaction))
	{
		goto out;
	}

	// Process connection and URI.
	if (!(ret = msc_process_connection(transaction, client_host,
		client_port, server_host, server_port)))
	{
		*err = msc_retval_process_connection;
		goto out;
	}
	if (retVal = processIntervention(transaction))
	{
		goto out;
	}

	if (!(ret = msc_process_uri(transaction, uri, http_method,
		http_version)))
	{
		*err = msc_retval_process_uri;
		goto out;
	}
	if (retVal = processIntervention(transaction))
	{
		goto out;
	}

	// Request headers.
	for(idx = 0; idx < reqHeaderSize; idx++)
	{
		reqHeaderKey = reqHeaderKeys[idx];
		reqHeaderVal = reqHeaderVals[idx];

		if (!(ret = msc_add_request_header(transaction, reqHeaderKey,
			reqHeaderVal)))
		{
			*err = msc_retval_add_request_header;
			goto out;
		}
		if (retVal = processIntervention(transaction)) {
			goto out;
		}
	}
	if (!(ret = msc_process_request_headers(transaction)))
	{
		*err = msc_retval_process_request_headers;
		goto out;
	}
	if (retVal = processIntervention(transaction))
	{
		goto out;
	}

	// Request body.
	if (!(ret = msc_append_request_body(transaction, reqBodyText,
		reqBodySize)))
	{
		*err = msc_retval_append_request_body;
		goto out;
	}
	if (retVal = processIntervention(transaction))
	{
		goto out;
	}
	if (!(ret = msc_process_request_body(transaction)))
	{
		*err = msc_retval_process_request_body;
		goto out;
	}
	if (retVal = processIntervention(transaction))
	{
		goto out;
	}

	// Logging.
	// XXX We need to remove it from here on future versions, it's better to
	// answer to envoy before logging
	if (!(ret = msc_process_logging(transaction)))
	{
		*err = msc_retval_process_logging;
		goto out;
	}

out:
	if (transaction != NULL)
	{
		msc_transaction_cleanup(transaction);
		transaction = NULL;
	}

	return retVal;
}

void CleanupModSecurity()
{
	if (rules != NULL)
	{
		msc_rules_cleanup( rules );
	}
	if (modsec != NULL)
	{
		msc_cleanup(modsec);
	}

	rules = NULL;
	modsec = NULL;
}

// Helper functions to store all core rule set file names in memory.
char **makeCharArray(int size)
{
	return calloc(sizeof(char*), size);
}

void freeCharArray(char **array, int size)
{
	int idx;
	for (idx = 0; idx < size; idx++)
	{
		free(array[idx]);
	}

	free(array);
}

void setArrayString(char **array, char *input, int idx)
{
	array[idx] = input;
}

// Free all interventions generated
void freeIntervention(ModSecurityIntervention *in)
{
	free(in->url);
	free(in->log);
	free(in);
}
