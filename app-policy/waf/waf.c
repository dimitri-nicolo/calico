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
	msc_retval_process_connection = -1,
	msc_retval_process_uri = -2,
	msc_retval_add_request_header = -3,
	msc_retval_process_request_headers = -4,
	msc_retval_append_request_body = -5,
	msc_retval_process_request_body = -6,
	msc_retval_process_logging = -7,

} enum_msc_retval;

// Private helper function to initialize ModSecurity.
static void initializeModSecurityImpl();

// Function prototype must match modsecurity.cc ModSecLogCb callback signature.
void CModSecurityLoggingCallback( void *referenceAPI, const void *ruleMessage )
{
    // Remove constness and coerce to char* to be compatible with Golang API.
    char *payload = (char *)ruleMessage;
    GoModSecurityLoggingCallback( payload );
}

// General public APIs.
void InitializeModSecurity()
{
    initializeModSecurityImpl();
}
static void initializeModSecurityImpl()
{
    modsec = msc_init();
    msc_set_log_cb( modsec, CModSecurityLoggingCallback );
    rules = msc_create_rules_set();
}

const char* LoadModSecurityCoreRuleSet( char *file )
{
    const char *error = NULL;
    char *error_message = NULL;
    if ( modsec == NULL )
    {
        initializeModSecurityImpl();
    }

    msc_rules_add_file( rules, file, &error );
    if ( error != NULL )
    {
        error_message = (char *)error;
    }

    error = NULL;
    return error_message;
}

int ProcessHttpRequest(
    char *id,
    char *uri,
    char *http_method,
    char *http_protocol,
    char *http_version,
    char *client_host,
    int client_port,
    char *server_host,
    int server_port,
    char **reqHeaderKeys,
    char **reqHeaderVals,
    int reqHeaderSize,
    char *reqBodyText,
    int reqBodySize
    )
{
    int retVal = 0;

    const char *reqHeaderKey = NULL;
    const char *reqHeaderVal = NULL;
    int index = 0;

    if ( modsec == NULL )
    {
        initializeModSecurityImpl();
    }

    Transaction *transaction = NULL;
    transaction = msc_new_transaction_with_id( modsec, rules, id, NULL );

    // Process connection and URI.
    retVal = msc_process_connection( transaction, client_host, client_port, server_host, server_port );
    if ( !retVal )
    {
        retVal = msc_retval_process_connection;
        goto out;
    }

    retVal = msc_process_uri( transaction, uri, http_method, http_version );
    if ( !retVal )
    {
        retVal = msc_retval_process_uri;
        goto out;
    }

    // Request headers.
    for( index = 0; index < reqHeaderSize; index++ )
    {
        reqHeaderKey = reqHeaderKeys[ index ];
        reqHeaderVal = reqHeaderVals[ index ];

        retVal = msc_add_request_header( transaction, reqHeaderKey, reqHeaderVal );
        if ( !retVal )
        {
            retVal = msc_retval_add_request_header;
            goto out;
        }
    }
    retVal = msc_process_request_headers( transaction );
    if ( !retVal )
    {
        retVal = msc_retval_process_request_headers;
        goto out;
    }

    // Request body.
    retVal = msc_append_request_body( transaction, reqBodyText, reqBodySize );
    if ( !retVal )
    {
        retVal = msc_retval_append_request_body;
        goto out;
    }
    retVal = msc_process_request_body( transaction );
    if ( !retVal )
    {
        retVal = msc_retval_process_request_body;
        goto out;
    }

    // Logging.
    retVal = msc_process_logging( transaction );
    if ( !retVal )
    {
        retVal = msc_retval_process_logging;
        goto out;
    }

    // Detects Mod Security intervention.
    ModSecurityIntervention intervention;
    intervention.status = 200;
    intervention.url = NULL;
    intervention.log = NULL;
    intervention.disruptive = 0;

    retVal = msc_intervention( transaction, &intervention );

out:
    if ( transaction != NULL )
    {
        msc_transaction_cleanup( transaction );
        transaction = NULL;
    }

    // Set errno for any potential ModSec return value failures to trigger Go err value to be returned upstream.
    errno = ( retVal < 0 ) ? retVal : 0;

    return retVal;
}

void CleanupModSecurity()
{
    if ( rules != NULL )
    {
        msc_rules_cleanup( rules );
    }
    if ( modsec != NULL )
    {
        msc_cleanup( modsec );
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
    int index;
    for (index = 0; index < size; index++)
    {
        free(array[index]);
    }

    free(array);
}
void setArrayString(char **array, char *input, int index)
{
    array[index] = input;
}

