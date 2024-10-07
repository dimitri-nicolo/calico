# Backend API for the Manager UI: tigera/ui-apis

This package contains a service (formerly known as "es-proxy") which provides collection of APIs designed to feed the Calico Enterprise and Calico Cloud web UI (a.k.a., the Manager UI).

The purpose of this service is to provide consistent APIs designed specifically to support the Manager UI (i.e., not for direct consumption by end users).

Note that while this service provides many APIs for the UI, not all APIs used by the Manager UI must be implemented here. While there may be reasons that your API wants to be in a separate service, keeping most simple APIs here helps simplify both product architecture and development.
