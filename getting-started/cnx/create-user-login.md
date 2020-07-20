---
title: Configure user authentication and log in
description: Create a user login using an authentication method, and log in to Calico Enterprise Manager with default roles. 
---

### Big picture

Configure an authentication method, create a user with access to the manager UI, and log in to the {{site.prodname}} Manager UI.

### Before you begin...

Make sure you have installed {{site.prodname}} using one of the [installation guides]({{site.baseurl}}/getting-started/) and have set up [access to the Manager UI]({{site.baseurl}}/getting-started/cnx/access-the-manager).

### Concepts

#### Authentication methods

The {{site.prodname}} Manager supports the following user authentication methods:

- **Token authentication (default)**: The user is a service account. When a service account is created, an associated secret is created that contains a signed bearer token for that service account.
- **OIDC authentication**: The user is managed outside of the cluster (typically, by the identity provider used by the OIDC authorization server.)
- **OAuth authentication**: The user is managed outside of the cluster (typically, by the identity provider used by the OAuth authorization server.) For OpenShift clusters, we recommend using OAuth authentication against OpenShift's internal OAuth server.
- **Basic authentication**: (for testing only) The user is a username. Note that basic authentication is not suitable for production environments.

#### Identity Providers, OIDC and OAuth

When configuring your cluster, you may be asked to provide information on the following concepts:

- **Identity Provider (IdP)**: A third party system to which user identity and authentication can be delegated.
- **Client Id**: The id that is shared between the IdP and an application for exchanging data.
- **Client Secret**: The secret associated with the `client id` can be used by server applications for the purpose of exchanging tokens.
- **Issuer URL**: The url where the IdP can be reached. The OIDC framework relies on conventions of which this URL is the basis.
- **Well known configuration**: The OIDC framework is designed to be flexible. The specifics of your IdP are then reflected in `well known configuration`, which is read by OIDC consumers.
- **Scopes**: When authenticating the IdP sometimes lists a number of scopes that the user consents to sharing with the application. Adding more scopes, can lead to sharing more metadata with the application.
- **Claims**: When you configure your IdP, you can configure claims. Every time your IdP issues a token for a valid user, these claims add some metadata as part of the token that the server can then use to tailor requests to the needs of a user. The most common example is to determine the username.

#### Cluster roles

In Kubernetes, **cluster roles** specify cluster scoped permissions and are bound to users via **cluster role bindings**.
Users must have appropriate RBAC to access resources in the UI. We provide the following roles by default to get started:

- **tigera-ui-user**: Allows basic UI access.
- **tigera-network-admin**: Allows UI access, plus the ability to create and modify resources, view compliance reports, and more.

If you would like additional roles, see this [document]({{site.baseurl}}/security/rbac-tiered-policies).

### How to
The page describes how to configure the following options:
- Configure the Calico Enterprise [authentication method](#configure-the-calico-enterprise-authentication-method)
- [Token-based authentication](#token-based-authentication)
- [OIDC authentication](#oidc-authentication)
- [OIDC authentication with prepopulated configuration](#oidc-authentication-with-prepopulated-configuration)
- [OAuth2 authentication](#oauth2-authentication)
- [basic authentication](#basic-authentication)
- [Kibana basic authentication](#kibana-basic-authentication)
- [Kibana OIDC authentication](#kibana-OIDC-authentication)

> Note: For OpenShift, replace `kubectl` in the commands below with `oc`.

#### Configure the Calico Enterprise authentication method

The {{site.prodname}} authentication method can be configured through the [Manager API resource]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.Manager) named `tigera-secure`.
If the authentication type is not specified, the default authentication method is `Token`.

Run one of the following commands to configure authentication for {{site.prodname}}.

**Token authentication (default)**

```bash
kubectl patch manager tigera-secure --type merge -p '{"spec": {"auth": {"type": "Token"}}}'
```

**OIDC authentication**

Provide your own values for `<oidc_auth_server>` and `<client_id>` and run:

```bash
kubectl patch manager tigera-secure --type merge -p '{"spec": {"auth": {"type": "OIDC", "authority": "<oidc_auth_server>", "clientID": "<client_id>"}}}'
```

If you are planning to use OIDC authentication with prepopulated configuration, keep `authority` value `<oidc_auth_server>` empty.

**OAuth2 authentication**

Provide your own values for `<oauth2_auth_server>` and `<client_id>` and run:

```bash
kubectl patch manager tigera-secure --type merge -p '{"spec": {"auth": {"type": "OAuth", "authority": "<oauth2_auth_server>", "clientID": "<client_id>"}}}'
```

**Basic authentication (for testing only)**

```bash
kubectl patch manager tigera-secure --type merge -p '{"spec": {"auth": {"type": "Basic"}}}'
```

#### Token-based authentication

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

#### OIDC authentication

1. Ensure that you have configured the required [kube-apiserver flags](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#configuring-the-api-server) for OIDC authentication.
1. Consult your OIDC identity provider's documentation to manage users.
1. Go to the {{site.prodname}} Manager UI. The OIDC authorization flow starts automatically.

#### OIDC authentication with prepopulated configuration

In cases where the IdP doesn't allow cross-origin HTTP requests, OIDC configuration can be prepopulated to support OIDC authentication flow.

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

   In above example, `<well-known-openid-configuration>` is the JSON response from the IdP for request to _/.well-known/openid-configuration_. Notice however that the `jwks_uri` value in `<well-known-openid-configuration>` should be set to `"/discovery/keys"`. For `<jwks-uri-configuration>`, use the JSON response from IdP for JWKS URI.
1. Go to the {{site.prodname}} Manager UI. The OIDC authorization flow starts automatically.

#### OAuth2 authentication

1. Consult your OAuth2 identity provider's documentation to manage users.
1. Go to the {{site.prodname}} Manager UI. The OAuth2 authorization flow starts automatically.

#### Login using basic authentication

Basic authentication is intended for testing purposes and is not suitable for production.
It has significant limitations—notably the Kubernetes API server must be restarted after making any changes.

1. Enable Kubernetes basic authentication by passing a static password file to the Kubernetes API server as discussed in the Kubernetes documentation.
1. Go to the {{site.prodname}} Manager UI and enter the username/password.

#### Kibana basic authentication	

Connect to Kibana with the `elastic` username. Use the following command to decode the password:	

{% raw %}
```	
kubectl -n tigera-elasticsearch get secret tigera-secure-es-elastic-user -o go-template='{{.data.elastic | base64decode}}' && echo
```
{% endraw %}
Once logged in, you can configure users and their privileges from the settings page.

#### Kibana OIDC authentication

Kibana can be configured to use your IdP. When you open the manager and click on the Kibana button, the user will be prompted by the IdP to login. Upon success, they will be redirected back to Kibana
and the username for Kibana will be extracted from the `usernameClaim` provided by the IdP. {{site.prodname}} is able to automatically translate {{site.prodname}} [RBAC permissions]({{site.baseurl}}/security/logs/rbac-elasticsearch)
 for the apiGroup `lma.tigera.io` to Elasticsearch User Role Mappings. 

1. Configure your [kube-apiserver to use OIDC](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#configuring-the-api-server) for OIDC authentication.

1. Apply the Authentication CR to your cluster to let the operator configure your login. This example demonstrates the email claim. 
   This means that from the JWT that your IdP creates, the email field is used as the username to bind privileges to. Make sure 
   that the `issuerURL` and `usernameClaim` match the configuration of your kube-apiserver. For more configuration options, 
   see the [Authentication resource]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.Authentication).

   ```
   apiVersion: operator.tigera.io/v1
   kind: Authentication
   metadata:
     name: tigera-secure
   spec:
     managerDomain: <domain-of-manager-ui>
     method: OIDC
     oidc:
       issuerURL: <your-IdP-issuer>
       usernameClaim: email
   ```

1. Apply the secret to your cluster with your OIDC credentials. By default, all scopes that are defined in the `well-known-configuration` are added by {{site.prodname}}.

   ```
   apiVersion: v1
   kind: Secret
   metadata:
     name: tigera-oidc-credentials
     namespace: tigera-operator
   data:
     clientID: <your-base64-clientid>
     clientSecret: <clientid-secret>
     requiredScopes: []
   ```

1. Give a user permissions to login to Kibana and to view the data. The following example gives full access to a user logged in using OIDC.

   ```
   apiVersion: rbac.authorization.k8s.io/v1
   kind: ClusterRole
   metadata:
     name: tigera-kibana-admin
   rules:
   - apiGroups:
     - lma.tigera.io
     resourceNames:
     - kibana_login
     - audit*
     - audit_ee
     - audit_kube
     - events
     - dns
     resources:
     - '*'
     verbs:
     - '*'
   ```

   ```
   kubectl create clusterrolebinding my-username-kibana-access --user=<username> --clusterrole=tigera-kibana-admin
   ```
   For more configuration options, see {{site.prodname}} [RBAC permissions]({{site.baseurl}}/security/logs/rbac-elasticsearch).
   
### Whitelisting {{site.prodname}} in your IdP
Most IdP's require Authorized Redirect URI's to be whitelisted, before the IdP will redirect users at the end of the OAuth flow to the {{site.prodname}} Manager or to Kibana. 
Similarly, most IdP's require authorizing browser (JavaScript) origins, since they are not able to provide a client secret. Please consult your IdP's documentation for authorizing your domain for the respective origins and destinations.

**Authorized JavaScript origins**
- Add the domain and port for your {{site.prodname}} Manager and Kibana

**Authorized redirect URIs**
- `https://<host>:<port>/login/oidc/callback`
- `https://<host>:<port>/tigera-kibana/api/security/oidc/callback`

### Above and beyond

- [Configure RBAC for tiered policies]({{site.baseurl}}/security/rbac-tiered-policies)
- [Configure Calico Enterprise RBAC for Elasticsearch]({{site.baseurl}}/security/logs/rbac-elasticsearch)
