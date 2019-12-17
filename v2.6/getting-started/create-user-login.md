---
title: Log in to Calico Enterprise Manager UI
redirect_from: latest/getting-started/create-user-login
---

### Big picture

Configure an authentication method, create a user with access to the manager UI, and log in to the {{site.prodname}} Manager UI.

### Before you begin...

Make sure you have installed {{site.prodname}} using one of the [installation guides](/{{page.version}}/getting-started/) and have setup
[access to the Manager UI](/{{page.version}}/getting-started/access-the-manager)

### Concepts

#### Authentication methods

The {{site.prodname}} Manager supports the following user authentication methods:

- **Token authentication (default)**: The user is a service account. When a service account is created, an associated secret is created that contains a signed bearer token for that service account.
- **OIDC authentication**: The user is managed outside of the cluster (typically, by the identity provider used by the OIDC authorization server.)
- **OAuth authentication**: The user is managed outside of the cluster (typically, by the identity provider used by the OAuth authorization server.) For OpenShift clusters, we recommend using OAuth authentication against OpenShift's internal OAuth server.
- **Basic authentication**: (for testing only) The user is a username. Note that basic authentication is not suitable for production environments.

#### Cluster roles

In Kubernetes, **cluster roles** specify cluster scoped permissions and are bound to users via **cluster role bindings**.
Users must have appropriate RBAC to access resources in the UI. We provide the following roles by default to get started:

- **tigera-ui-user**: Allows basic UI access.
- **tigera-network-admin**: Allows UI access, plus the ability to create and modify resources, view compliance reports, and more.

If you would like additional roles, see this [document]({{site.url}}/{{page.version}}/reference/cnx/rbac-tiered-policies#example-fine-grained-permissions).

### How to

- [Configure the Calico Enterprise authentication method](#configure-the-calico-enterprise-authentication-method)
- [Create a user and login using token-based authentication](#create-a-user-and-login-using-token-based-authentication)
- [Create a user and login using OIDC authentication](#create-a-user-and-login-using-oidc-authentication)
- [Create a user and login using OIDC authentication with prepopulated configuration](#create-a-user-and-login-using-oidc-authentication-with-prepopulated-configuration)
- [Create a user and login using OAuth2 authentication](#create-a-user-and-login-using-oauth2-authentication)
- [Create a user and login using basic authentication](#create-a-user-and-login-using-basic-authentication)

> Note: For OpenShift, replace `kubectl` in the commands below with `oc`.

#### Configure the Calico Enterprise authentication method

The {{site.prodname}} authentication method can be configured through the [Manager API resource]({{site.url}}/{{page.version}}/reference/installation/api#operator.tigera.io/v1.Manager) named `calico-enterprise`.
If the authentication type is not specified, the default authentication method is `Token`.

Run one of the following commands to configure authentication for {{site.prodname}}.

**Token authentication (default)**

```bash
kubectl patch manager calico-enterprise --type merge -p '{"spec": {"auth": {"type": "Token"}}}'
```

**OIDC authentication**

Provide your own values for `<oidc_auth_server>` and `<client_id>` and run:

```bash
kubectl patch manager calico-enterprise --type merge -p '{"spec": {"auth": {"type": "OIDC", "authority": "<oidc_auth_server>", "clientID": "<client_id>"}}}'
```

If you are planning to use OIDC authentication with prepopulated configuration, keep `authority` value `<oidc_auth_server>` empty.

**OAuth2 authentication**

Provide your own values for `<oauth2_auth_server>` and `<client_id>` and run:

```bash
kubectl patch manager calico-enterprise --type merge -p '{"spec": {"auth": {"type": "OAuth", "authority": "<oauth2_auth_server>", "clientID": "<client_id>"}}}'
```

**Basic authentication (for testing only)**

```bash
kubectl patch manager calico-enterprise --type merge -p '{"spec": {"auth": {"type": "Basic"}}}'
```

#### Create a user and login using token-based authentication

First, create a service account in the desired namespace:

```bash
kubectl create sa <user> -n <namespace>
```

Give the service account permissions to access the {{site.prodname}} Manager UI, and a {{site.prodname}} cluster role:

```bash
kubectl create clusterrolebinding <binding_name> --clusterrole <role_name> --serviceaccount <namespace>:<service_account>
```

where:
- **binding_name** is a descriptive name for the rolebinding.
- **role_name** is one of the default cluster roles (or a custom cluster role) specifying {{site.prodname}} UI permissions.
- **namespace** is the service account's namespace.
- **service_account** is the service account that the permissions are being associated with.

For example, the following command gives the service account jane in the default namespace network admin permissions:

```bash
kubectl create clusterrolebinding jane-access --clusterrole tigera-network-admin --serviceaccount default:jane
```

Next, get the token from the service account.
Using the running example of a service account named jane in the default namespace:

```bash
{% raw %}kubectl get secret $(kubectl get serviceaccount jane -o jsonpath='{range .secrets[*]}{.name}{"\n"}{end}' | grep token) -o go-template='{{.data.token | base64decode}}' && echo{% endraw %}
```

Now that we have the token, we can proceed to login! Go to the {{site.prodname}} UI and submit the token.

#### Create a user and login using OIDC authentication

1. Ensure that you have configured the required [kube-apiserver flags](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#configuring-the-api-server) for OIDC authentication.
1. Consult your OIDC identity provider's documentation to manage users.
1. Go to the {{site.prodname}} Manager UI. The OIDC authorization flow starts automatically.

#### Create a user and login using OIDC authentication with prepopulated configuration

In cases where OIDC Identity Provider (IdP) doesn't allow cross-origin HTTP requests, OIDC configuration can be prepopulated to support OIDC authentication flow.

1. Consult your OIDC identity provider's documentation to manage users.
1. Make sure OIDC authority is set to empty value.
1. Set up configuration under `tigera-operator` namespace, populating OIDC configurations (e.g. authorization and token endpoints, JWK keys etc.). For example:

   ```
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: tigera-manager-oidc-config
     namespace: tigera-operator
   data:
     openid-configuration: |
       <well-known-openid-configuration>
       ...
       "jwks_uri": "/discovery/keys",
       ...
     keys: |
       <jwks-uri-configuration>
   ```

   In above example, `<well-known-openid-configuration>` is the JSON response from IdP for request to _/.well-known/openid-configuration_. Notice however that the `jwks_uri` value in `<well-known-openid-configuration>` should be set to `"/discovery/keys"`. For `<jwks-uri-configuration>`, use the JSON response from IdP for JWKS URI.
1. Go to the {{site.prodname}} Manager UI. The OIDC authorization flow starts automatically.

#### Create a user and login using OIDC authentication with prepopulated configuration

In cases where OIDC Identity Provider (IdP) doesn't allow cross-origin HTTP requests, OIDC configuration can be prepopulated to support OIDC authentication flow.

1. Consult your OIDC identity provider's documentation to manage users.
1. Make sure OIDC authority is set to empty value.
1. Set up configuration under `tigera-operator` namespace, populating OIDC configurations (e.g. authorization and token endpoints, JWK keys etc.). For example:

   ```
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: tigera-manager-oidc-config
     namespace: tigera-operator
   data:
     openid-configuration: |
       <well-known-openid-configuration>
       ...
       "jwks_uri": "/discovery/keys",
       ...
     keys: |
       <jwks-uri-configuration>
   ```

   In above example, `<well-known-openid-configuration>` is the JSON response from IdP for request to _/.well-known/openid-configuration_. Notice however that the `jwks_uri` value in `<well-known-openid-configuration>` should be set to `"/discovery/keys"`. For `<jwks-uri-configuration>`, use the JSON response from IdP for JWKS URI.
1. Go to the {{site.prodname}} Manager UI. The OIDC authorization flow starts automatically.

#### Create a user and login using OAuth2 authentication

1. Consult your OAuth2 identity provider's documentation to manage users.
1. Go to the {{site.prodname}} Manager UI. The OAuth2 authorization flow starts automatically.

#### Create a user and login using basic authentication

Basic authentication is intended for testing purposes and is not suitable for production.
It has significant limitations—notably the Kubernetes API server must be restarted after making any changes.

1. Enable Kubernetes basic authentication by passing a static password file to the Kubernetes API server as discussed in the Kubernetes documentation.
1. Go to the {{site.prodname}} Manager UI and enter the username/password.

### Above and beyond

- [Fine-grained RBAC permissions]({{site.url}}/{{page.version}}/reference/cnx/rbac-tiered-policies#example-fine-grained-permissions)
