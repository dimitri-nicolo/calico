// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

#include "waf.h"
#include "_cgo_export.h"
#include "modsecurity/modsecurity.h"
#include "modsecurity/rules_set.h"
#include "modsecurity/transaction.h"
#include "modsecurity/intervention.h"

ModSecurity *modsec = NULL;
RulesSet *rules = NULL;

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

int ProcessHttpRequest( char *id, char *uri, char *http_method, char *http_protocol, char *http_version, char *client_host, int client_port, char *server_host, int server_port )
{
    int detection = 0;
    if ( modsec == NULL )
    {
        initializeModSecurityImpl();
    }

    Transaction *transaction = NULL;
    transaction = msc_new_transaction_with_id( modsec, rules, id, NULL );
    msc_process_connection( transaction, client_host, client_port, server_host, server_port );
    msc_process_uri( transaction, uri, http_protocol, http_version );
    msc_process_request_headers( transaction );
    msc_process_request_body( transaction );

    ModSecurityIntervention intervention;
    intervention.status = 200;
    intervention.url = NULL;
    intervention.log = NULL;
    intervention.disruptive = 0;

    detection = msc_intervention( transaction, &intervention );
    if ( transaction != NULL )
    {
        msc_transaction_cleanup( transaction );
        transaction = NULL;
    }

    return detection;
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

