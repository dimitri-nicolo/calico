// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

// Important: <stdlib.h> header file required for Golang statements e.g. defer C.free(unsafe.Pointer(API))
#include <stdlib.h>

void InitializeModSecurity();
const char* LoadModSecurityCoreRuleSet( char *file );
int ProcessHttpRequest( char *id, char *uri, char *http_method, char *http_protocol, char *http_version, char *client_host, int client_port, char *server_host, int server_port );
void CleanupModSecurity();

