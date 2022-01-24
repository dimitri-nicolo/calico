// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

// ModSecurity logging callback infrastructure APIs.
void InvokeModSecurityLoggingCallback( ModSecurityLoggingCallbackFunctionPointer func, char *payload )
{
    // Invoke Golang callback with ModSecurity logging payload i.e. C => Go code invocation.
    func( payload );
}
// Function prototype must match modsecurity.cc ModSecLogCb callback signature.
void CModSecurityLoggingCallback( void *referenceAPI, const void *ruleMessage )
{
    // Remove constness and coerce to char* to be compatible with Golang API.
    char *payload = (char *)ruleMessage;
    InvokeModSecurityLoggingCallback( &GoModSecurityLoggingCallback, payload );
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

int LoadModSecurityCoreRuleSet(char **array, int size)
{
    int index = 0;
    const char *file;
    const char *error = NULL;
    if ( modsec == NULL )
    {
        initializeModSecurityImpl();
    }

    for( index = 0; index < size; index++ )
    {
        file = array[ index ];
        msc_rules_add_file( rules, file, &error );
        if ( error != NULL )
        {
            break;
        }
    }

    return index;
}

int ProcessHttpRequest( char *id, char *uri, char *http_method, char *http_protocol, char *http_version, char *client_host, int client_port, char *server_host, int server_port )
{
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
    return msc_intervention( transaction, &intervention );
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
char **makeCharArray( int size )
{
    return calloc( sizeof( char* ) , size );
}
void freeCharArray( char **array, int size )
{
    int index;
    for ( index = 0; index < size; index++ )
    {
        free( array[ index ] );
    }

    free( array );
}
void setArrayString( char **array, char *filename, int index)
{
    array[ index ] = filename;
}
