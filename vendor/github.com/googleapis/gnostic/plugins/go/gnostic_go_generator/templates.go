// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

func templates() map[string]string {
	return map[string]string{ 
        "client.go": "Ly8gR0VORVJBVEVEIEZJTEU6IERPIE5PVCBFRElUIQoKcGFja2FnZSB7ey5SZW5kZXJlci5QYWNrYWdlfX0KCmltcG9ydCAoCiAgImJ5dGVzIgogICJlcnJvcnMiCiAgImVuY29kaW5nL2pzb24iCiAgImZtdCIKICAibmV0L2h0dHAiCiAgInN0cmluZ3MiCikKICAKLy8gQVBJIGNsaWVudCByZXByZXNlbnRhdGlvbi4KdHlwZSBDbGllbnQgc3RydWN0IHsKCXNlcnZpY2Ugc3RyaW5nCn0gCgovLyBDcmVhdGUgYW4gQVBJIGNsaWVudC4KZnVuYyBOZXdDbGllbnQoc2VydmljZSBzdHJpbmcpICpDbGllbnQgewoJY2xpZW50IDo9ICZDbGllbnR7fQoJY2xpZW50LnNlcnZpY2UgPSBzZXJ2aWNlCglyZXR1cm4gY2xpZW50Cn0KCi8vLXt7cmFuZ2UgLlJlbmRlcmVyLk1ldGhvZHN9fQp7e2NvbW1lbnRGb3JUZXh0IC5EZXNjcmlwdGlvbn19Ci8vLXt7aWYgZXEgLlJlc3VsdFR5cGVOYW1lICIifX0KZnVuYyAoY2xpZW50ICpDbGllbnQpIHt7LkNsaWVudE5hbWV9fSh7e3BhcmFtZXRlckxpc3QgLn19KSAoZXJyIGVycm9yKSB7Ci8vLXt7ZWxzZX19CmZ1bmMgKGNsaWVudCAqQ2xpZW50KSB7ey5DbGllbnROYW1lfX0oe3twYXJhbWV0ZXJMaXN0IC59fSkgKHJlc3VsdCAqe3suUmVzdWx0VHlwZU5hbWV9fSwgZXJyIGVycm9yKSB7Ci8vLXt7ZW5kfX0KCXBhdGggOj0gY2xpZW50LnNlcnZpY2UgKyAie3suUGF0aH19IgoJLy8te3tpZiBoYXNQYXJhbWV0ZXJzIC59fQoJLy8te3tyYW5nZSAuUGFyYW1ldGVyc1R5cGUuRmllbGRzfX0JCgkvLy17e2lmIGVxIC5Qb3NpdGlvbiAicGF0aCJ9fQoJcGF0aCA9IHN0cmluZ3MuUmVwbGFjZShwYXRoLCAieyIgKyAie3suSlNPTk5hbWV9fSIgKyAifSIsIGZtdC5TcHJpbnRmKCIldiIsIHt7LkpTT05OYW1lfX0pLCAxKQoJLy8te3tlbmR9fQoJLy8te3tlbmR9fQoJLy8te3tlbmR9fQoJLy8te3tpZiBlcSAuTWV0aG9kICJQT1NUIn19Cglib2R5IDo9IG5ldyhieXRlcy5CdWZmZXIpCglqc29uLk5ld0VuY29kZXIoYm9keSkuRW5jb2RlKHt7Ym9keVBhcmFtZXRlck5hbWUgLn19KQoJcmVxLCBlcnIgOj0gaHR0cC5OZXdSZXF1ZXN0KCJ7ey5NZXRob2R9fSIsIHBhdGgsIGJvZHkpCgkvLy17e2Vsc2V9fQoJcmVxLCBlcnIgOj0gaHR0cC5OZXdSZXF1ZXN0KCJ7ey5NZXRob2R9fSIsIHBhdGgsIG5pbCkKCS8vLXt7ZW5kfX0KCWlmIGVyciAhPSBuaWwgewoJCXJldHVybgoJfQoJcmVzcCwgZXJyIDo9IGh0dHAuRGVmYXVsdENsaWVudC5EbyhyZXEpCglpZiBlcnIgIT0gbmlsIHsKCQlyZXR1cm4KCX0KCWlmIHJlc3AuU3RhdHVzQ29kZSA9PSAyMDAgewoJCWRlZmVyIHJlc3AuQm9keS5DbG9zZSgpCgkJLy8te3tpZiBuZSAuUmVzdWx0VHlwZU5hbWUgIiJ9fQoJCWRlY29kZXIgOj0ganNvbi5OZXdEZWNvZGVyKHJlc3AuQm9keSkKCQlyZXN1bHQgPSAme3suUmVzdWx0VHlwZU5hbWV9fXt9CgkJZGVjb2Rlci5EZWNvZGUocmVzdWx0KQoJCS8vLXt7ZW5kfX0KCX0gZWxzZSB7CgkJZXJyID0gZXJyb3JzLk5ldyhyZXNwLlN0YXR1cykKCX0KCXJldHVybgp9CgovLy17e2VuZH19CgovLyByZWZlciB0byBpbXBvcnRlZCBwYWNrYWdlcyB0aGF0IG1heSBvciBtYXkgbm90IGJlIHVzZWQgaW4gZ2VuZXJhdGVkIGNvZGUKZnVuYyBmb3JjZWRfcGFja2FnZV9yZWZlcmVuY2VzKCkgewoJXyA9IG5ldyhieXRlcy5CdWZmZXIpCglfID0gZm10LlNwcmludGYoIiIpCglfID0gc3RyaW5ncy5TcGxpdCgiIiwiIikKfQ==",
        "provider.go": "Ly8gR0VORVJBVEVEIEZJTEU6IERPIE5PVCBFRElUIQoKcGFja2FnZSB7ey5SZW5kZXJlci5QYWNrYWdlfX0KCi8vIFRvIGNyZWF0ZSBhIHNlcnZlciwgZmlyc3Qgd3JpdGUgYSBjbGFzcyB0aGF0IGltcGxlbWVudHMgdGhpcyBpbnRlcmZhY2UuCi8vIFRoZW4gcGFzcyBhbiBpbnN0YW5jZSBvZiBpdCB0byBJbml0aWFsaXplKCkuCnR5cGUgUHJvdmlkZXIgaW50ZXJmYWNlIHsKLy8te3tyYW5nZSAuUmVuZGVyZXIuTWV0aG9kc319CgovLyBQcm92aWRlcgp7e2NvbW1lbnRGb3JUZXh0IC5EZXNjcmlwdGlvbn19Ci8vLXt7aWYgaGFzUGFyYW1ldGVycyAufX0KLy8te3tpZiBoYXNSZXNwb25zZXMgLn19CiAge3suUHJvY2Vzc29yTmFtZX19KHBhcmFtZXRlcnMgKnt7LlBhcmFtZXRlcnNUeXBlTmFtZX19LCByZXNwb25zZXMgKnt7LlJlc3BvbnNlc1R5cGVOYW1lfX0pIChlcnIgZXJyb3IpCi8vLXt7ZWxzZX19CiAge3suUHJvY2Vzc29yTmFtZX19KHBhcmFtZXRlcnMgKnt7LlBhcmFtZXRlcnNUeXBlTmFtZX19KSAoZXJyIGVycm9yKQovLy17e2VuZH19Ci8vLXt7ZWxzZX19Ci8vLXt7aWYgaGFzUmVzcG9uc2VzIC59fQogIHt7LlByb2Nlc3Nvck5hbWV9fShyZXNwb25zZXMgKnt7LlJlc3BvbnNlc1R5cGVOYW1lfX0pIChlcnIgZXJyb3IpCi8vLXt7ZWxzZX19CiAge3suUHJvY2Vzc29yTmFtZX19KCkgKGVyciBlcnJvcikKLy8te3tlbmR9fQovLy17e2VuZH19CQovLy17e2VuZH19Cn0K",
        "server.go": "Ly8gR0VORVJBVEVEIEZJTEU6IERPIE5PVCBFRElUIQoKcGFja2FnZSB7ey5SZW5kZXJlci5QYWNrYWdlfX0KCmltcG9ydCAoCgkiZW5jb2RpbmcvanNvbiIKCSJlcnJvcnMiCgkibmV0L2h0dHAiCgkic3RyY29udiIKCgkiZ2l0aHViLmNvbS9nb3JpbGxhL211eCIKKQoKZnVuYyBpbnRWYWx1ZShzIHN0cmluZykgKHYgaW50NjQpIHsKCXYsIF8gPSBzdHJjb252LlBhcnNlSW50KHMsIDEwLCA2NCkKCXJldHVybiB2Cn0KCi8vIFRoaXMgcGFja2FnZS1nbG9iYWwgdmFyaWFibGUgaG9sZHMgdGhlIHVzZXItd3JpdHRlbiBQcm92aWRlciBmb3IgQVBJIHNlcnZpY2VzLgovLyBTZWUgdGhlIFByb3ZpZGVyIGludGVyZmFjZSBmb3IgZGV0YWlscy4KdmFyIHByb3ZpZGVyIFByb3ZpZGVyCgovLyBUaGVzZSBoYW5kbGVycyBzZXJ2ZSBBUEkgbWV0aG9kcy4KLy8te3tyYW5nZSAuUmVuZGVyZXIuTWV0aG9kc319CgovLyBIYW5kbGVyCnt7Y29tbWVudEZvclRleHQgLkRlc2NyaXB0aW9ufX0KZnVuYyB7ey5IYW5kbGVyTmFtZX19KHcgaHR0cC5SZXNwb25zZVdyaXRlciwgciAqaHR0cC5SZXF1ZXN0KSB7Cgl2YXIgZXJyIGVycm9yCgkvLy17e2lmIGhhc1BhcmFtZXRlcnMgLn19CgkvLyBpbnN0YW50aWF0ZSB0aGUgcGFyYW1ldGVycyBzdHJ1Y3R1cmUKCXZhciBwYXJhbWV0ZXJzIHt7LlBhcmFtZXRlcnNUeXBlTmFtZX19CgkvLy17e2lmIGVxIC5NZXRob2QgIlBPU1QifX0KCS8vIGRlc2VyaWFsaXplIHJlcXVlc3QgZnJvbSBwb3N0IGRhdGEKCWRlY29kZXIgOj0ganNvbi5OZXdEZWNvZGVyKHIuQm9keSkKCWVyciA9IGRlY29kZXIuRGVjb2RlKCZwYXJhbWV0ZXJzLnt7Ym9keVBhcmFtZXRlckZpZWxkTmFtZSAufX0pCglpZiBlcnIgIT0gbmlsIHsKCQl3LldyaXRlSGVhZGVyKGh0dHAuU3RhdHVzQmFkUmVxdWVzdCkKCQl3LldyaXRlKFtdYnl0ZShlcnIuRXJyb3IoKSArICJcbiIpKQoJCXJldHVybgoJfQoJLy8te3tlbmR9fQoJLy8gZ2V0IHJlcXVlc3QgZmllbGRzIGluIHBhdGggYW5kIHF1ZXJ5IHBhcmFtZXRlcnMKCS8vLXt7aWYgaGFzUGF0aFBhcmFtZXRlcnMgLn19Cgl2YXJzIDo9IG11eC5WYXJzKHIpCgkvLy17e2VuZH19CgkvLy17e2lmIGhhc0Zvcm1QYXJhbWV0ZXJzIC59fQoJci5QYXJzZUZvcm0oKQoJLy8te3tlbmR9fQoJLy8te3tyYW5nZSAuUGFyYW1ldGVyc1R5cGUuRmllbGRzfX0JCgkvLy17e2lmIGVxIC5Qb3NpdGlvbiAicGF0aCJ9fQoJaWYgdmFsdWUsIG9rIDo9IHZhcnNbInt7LkpTT05OYW1lfX0iXTsgb2sgewoJCXBhcmFtZXRlcnMue3suTmFtZX19ID0gaW50VmFsdWUodmFsdWUpCgl9CgkvLy17e2VuZH19CQoJLy8te3tpZiBlcSAuUG9zaXRpb24gImZvcm1kYXRhIn19CglpZiBsZW4oci5Gb3JtWyJ7ey5KU09OTmFtZX19Il0pID4gMCB7CgkJcGFyYW1ldGVycy57ey5OYW1lfX0gPSBpbnRWYWx1ZShyLkZvcm1bInt7LkpTT05OYW1lfX0iXVswXSkKCX0KCS8vLXt7ZW5kfX0KCS8vLXt7ZW5kfX0KCS8vLXt7ZW5kfX0KCS8vLXt7aWYgaGFzUmVzcG9uc2VzIC59fQkKCS8vIGluc3RhbnRpYXRlIHRoZSByZXNwb25zZXMgc3RydWN0dXJlCgl2YXIgcmVzcG9uc2VzIHt7LlJlc3BvbnNlc1R5cGVOYW1lfX0KCS8vLXt7ZW5kfX0KCS8vIGNhbGwgdGhlIHNlcnZpY2UgcHJvdmlkZXIJCgkvLy17e2lmIGhhc1BhcmFtZXRlcnMgLn19CgkvLy17e2lmIGhhc1Jlc3BvbnNlcyAufX0KCWVyciA9IHByb3ZpZGVyLnt7LlByb2Nlc3Nvck5hbWV9fSgmcGFyYW1ldGVycywgJnJlc3BvbnNlcykKCS8vLXt7ZWxzZX19CgllcnIgPSBwcm92aWRlci57ey5Qcm9jZXNzb3JOYW1lfX0oJnBhcmFtZXRlcnMpCgkvLy17e2VuZH19CgkvLy17e2Vsc2V9fQoJLy8te3tpZiBoYXNSZXNwb25zZXMgLn19CgllcnIgPSBwcm92aWRlci57ey5Qcm9jZXNzb3JOYW1lfX0oJnJlc3BvbnNlcykKCS8vLXt7ZWxzZX19CgllcnIgPSBwcm92aWRlci57ey5Qcm9jZXNzb3JOYW1lfX0oKQoJLy8te3tlbmR9fQoJLy8te3tlbmR9fQkKCWlmIGVyciA9PSBuaWwgewoJLy8te3sgaWYgaGFzUmVzcG9uc2VzIC59fQoJCS8vLXt7IGlmIC5SZXNwb25zZXNUeXBlIHwgaGFzRmllbGROYW1lZE9LIH19CQkKCQlpZiByZXNwb25zZXMuT0sgIT0gbmlsIHsKCQkJLy8gd3JpdGUgdGhlIG5vcm1hbCByZXNwb25zZQoJCQllbmNvZGVyIDo9IGpzb24uTmV3RW5jb2Rlcih3KQoJCQllbmNvZGVyLkVuY29kZShyZXNwb25zZXMuT0spCgkJCXJldHVybgoJCX0gCgkJLy8te3tlbmR9fQoJCS8vLXt7IGlmIC5SZXNwb25zZXNUeXBlIHwgaGFzRmllbGROYW1lZERlZmF1bHQgfX0JCQoJCWlmIHJlc3BvbnNlcy5EZWZhdWx0ICE9IG5pbCB7CgkJCXcuV3JpdGVIZWFkZXIoaW50KHJlc3BvbnNlcy5EZWZhdWx0LkNvZGUpKQoJCQl3LldyaXRlKFtdYnl0ZShyZXNwb25zZXMuRGVmYXVsdC5NZXNzYWdlICsgIlxuIikpCgkJCXJldHVybgoJCX0KCQkvLy17e2VuZH19CgkvLy17e2VuZH19Cgl9IGVsc2UgewoJCXcuV3JpdGVIZWFkZXIoaHR0cC5TdGF0dXNJbnRlcm5hbFNlcnZlckVycm9yKQoJCXcuV3JpdGUoW11ieXRlKGVyci5FcnJvcigpICsgIlxuIikpCgkJcmV0dXJuCgl9Cn0KLy8te3tlbmR9fQoKLy8gSW5pdGlhbGl6ZSB0aGUgQVBJIHNlcnZpY2UuCmZ1bmMgSW5pdGlhbGl6ZShwIFByb3ZpZGVyKSB7Cglwcm92aWRlciA9IHAKCXZhciByb3V0ZXIgPSBtdXguTmV3Um91dGVyKCl7e3JhbmdlIC5SZW5kZXJlci5NZXRob2RzfX0KCXJvdXRlci5IYW5kbGVGdW5jKCJ7ey5QYXRofX0iLCB7ey5IYW5kbGVyTmFtZX19KS5NZXRob2RzKCJ7ey5NZXRob2R9fSIpe3tlbmR9fQoJaHR0cC5IYW5kbGUoIi8iLCByb3V0ZXIpCn0KCi8vIFByb3ZpZGUgdGhlIEFQSSBzZXJ2aWNlIG92ZXIgSFRUUC4KZnVuYyBTZXJ2ZUhUVFAoYWRkcmVzcyBzdHJpbmcpIGVycm9yIHsKCWlmIHByb3ZpZGVyID09IG5pbCB7CgkJcmV0dXJuIGVycm9ycy5OZXcoIlVzZSB7ey5SZW5kZXJlci5QYWNrYWdlfX0uSW5pdGlhbGl6ZSgpIHRvIHNldCBhIHNlcnZpY2UgcHJvdmlkZXIuIikKCX0KCXJldHVybiBodHRwLkxpc3RlbkFuZFNlcnZlKGFkZHJlc3MsIG5pbCkKfQo=",
        "types.go": "Ly8gR0VORVJBVEVEIEZJTEU6IERPIE5PVCBFRElUIQoKcGFja2FnZSB7ey5SZW5kZXJlci5QYWNrYWdlfX0KCi8vIFR5cGVzIHVzZWQgYnkgdGhlIEFQSS4KLy8te3tyYW5nZSAuUmVuZGVyZXIuVHlwZXN9fQoKLy8te3tpZiBlcSAuS2luZCAic3RydWN0In19CnR5cGUge3suTmFtZX19IHN0cnVjdCB7IAovLy17e3JhbmdlIC5GaWVsZHN9fQogIHt7Lk5hbWV9fSB7e2dvVHlwZSAuVHlwZX19e3tpZiBuZSAuSlNPTk5hbWUgIiJ9fSBganNvbjoie3suSlNPTk5hbWV9fSJgCi8vLXt7ZW5kfX0KLy8te3tlbmR9fQp9Ci8vLXt7ZWxzZX19CnR5cGUge3suTmFtZX19IHt7LktpbmR9fQovLy17e2VuZH19CgovLy17e2VuZH19",
    }
}