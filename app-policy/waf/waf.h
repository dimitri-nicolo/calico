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

// Important: <stdlib.h> header file required for Golang statements e.g. defer C.free(unsafe.Pointer(API))
#include <stdlib.h>

// ModSecurity logging callback infrastructure APIs.
typedef void( *ModSecurityLoggingCallbackFunctionPointer )( char * );
extern void InvokeModSecurityLoggingCallback( ModSecurityLoggingCallbackFunctionPointer, char * );

void InitializeModSecurity();
int LoadModSecurityCoreRuleSet( char **array, int size );
int ProcessHttpRequest( char *id, char *uri, char *http_method, char *http_protocol, char *http_version, char *client_host, int client_port, char *server_host, int server_port );
void CleanupModSecurity();

// Helper functions to store all core rule set file names in memory.
char **makeCharArray( int size );
void freeCharArray( char **array, int size );
void setArrayString( char **array, char *filename, int index );
