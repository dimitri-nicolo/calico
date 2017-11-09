---
title: User authentication in CNX Manager
---

This document describes the authentication methods supported by Tigera CNX
Manager, and how to set them up.

After setting up authentication, see the [RBAC on tiered policies](rbac-tiered-policies)
document for information on how to control what resources each user can access.

CNX doesn't have its own authentication and authorization system, it delegates
to Kubernetes.  Detailed information on how to set up each authentication
method is provided by the [Kubernetes authentication guide](https://kubernetes.io/docs/admin/authentication/).

The CNX Manager web interface allows users to select the authentication method
to use, but Kubernetes must be configured to support the chosen method.
Select "Menu" on any sign-in page to change the login method in the web application.

### Google login

Google login allows users to log in using their Google accounts.  To use this
method, you need to [setup Google OAuth 2.0](https://developers.google.com/identity/protocols/OpenIDConnect),
and [configure the Kubernetes API server to use it](https://kubernetes.io/docs/admin/authentication/#configuring-the-api-server).

To use google login, the web server component of Tigera CNX Manager needs
to have a well known DNS name at which users access the application.  The sample
manifests in this documentation create a `NodePort` for the web server
serving on port 30003, but you may wish to set up connectivity differently.

1. [Setup Google OAuth 2.0](https://developers.google.com/identity/protocols/OpenIDConnect).
   - Ensure the redirect URIs are set to `https://<CNX Manager name>:<port>/login/oidc/callback`.
   - Note down the client ID.

2. [Configure the Kubernetes API server to use it](https://kubernetes.io/docs/admin/authentication/#configuring-the-api-server).
   - Use the client ID from above.

3. Configure Tigera CNX Manager to use it
   - Set the client ID in the `tigera-cnx-manager-web-config` ConfigMap (referenced
     in the installation instructions).

Configure Kubernetes RBAC bindings depending on the settings used for
`--oidc-username/groups-claim` and `--oidc-username/groups-prefix` on the API server.  When using this login method, be aware that most people have Google
accounts, so the system:authenticated group will include anybody who can
reach the cluster.

### Basic auth (username / password)

Basic auth allows users to configure Kubernetes with a list of username/passwords.
It is intended for testing purposes, and has significant limitations - notably
the Kubernetes API server must be restarted after making any changes.

Consult the [Kubernetes docs](https://kubernetes.io/docs/admin/authentication/#static-password-file)
to configure this login mode, and select the "Login via username and password"
option in the CNX Manager web interface.

Configure Kubernetes RBAC bindings using the username and groups defined in the
basic auth csv file.

### Static tokens

The "Login via static token" option tells CNX Manager to pass the token through
to Kubernetes.  It has similar limitations to basic auth.  The [Kubernetes docs](https://kubernetes.io/docs/admin/authentication/#static-token-file)
describe how to set up this method.

Like basic auth, RBAC bindings use the username and groups defined in the token
file.
