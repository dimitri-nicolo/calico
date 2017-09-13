# Authentication and Authorization

Since the Aggregated API Server delegates AuthN/Z to the Core API Server, we expect to perform none of the AuthN setup as part of a real world deployment scenario of our API Server. The UI login workflow is laid out based on the assumption that a real world Kubernetes Cluster Core API Server is most likely to be already setup with either Basic Auth or more sophisticated OAuth based OIDC authentication workflow.

For the purposes of demo/testing, following are the ways in which the Core API Server can be configured to do Authentication.

## Setting up OIDC

Assuming you have a running Kubernetes cluster (based on instructions as laid out in the README.md), the following steps can be used to configure OIDC on the Core API Server using Google as the Identity Provider. 

### Create a Google API Console project and obtain a Client ID. 
1. Go to https://console.developers.google.com/projectselector/apis/library.
2. Create/Select a project.
3. In the sidebar under APIs & Services, select Credentials, then select the OAuth consent screen tab.
4. Choose an Email Address, specify a Product Name, and submit Save.
5. In the Credentials tab, click on Create Credentials, and choose OAuth client ID from the drop-down.
6. Under Application type, select Other.
7. From the resulting OAuth client dialog box, copy the Client ID. The Client ID lets your app access enabled Google APIs.
8. Download the client secret JSON file of the credentials.

### Setting up a Kubernetes cluster
1. After initializing the master instance, you need to update the kube api server arguments in the manifest /etc/kubernetes/manifests/kube-apiserver.yaml.
```
sed -i "/- apiserver/a\    - --oidc-issuer-url=https://accounts.google.com\n    - --oidc-username-claim=email\n    - --oidc-client-id=<FILL_IN_THE_CLIENT_ID_FROM_CONSOLE_PROJECT" /etc/kubernetes/manifests/kube-apiserver.yaml
```

### For Kubectl access through OIDC
1. Install the helper on the client machine. Run the following command:
```
go get github.com/micahhausler/k8s-oidc-helper
```

2. Generate a user's credentials for kube config. Run the following command:
```
k8s-oidc-helper -c path/to/client_secret_[CLIENT_ID].json
```
The above is the secret JSON downloaded as part of Step 8.

This command should provide you with a URL. Paste it in a browser. It gives you a token in the browser. Copy it and paste to the terminal for k8s-oidc-helper. The output of the command should look as follows:
```
users:
- name: name@example.com
  user:
    auth-provider:
      config:
        client-id: 32934980234312-9ske1sskq89423480922scag3hutrv7.apps.googleusercontent.com
        client-secret: ZdyKxYW-tCzuRWwB3l665cLY
        id-token: eyJhbGciOiJSUzI19fvTKfPraZ7yzn.....HeLnf26MjA
        idp-issuer-url: https://accounts.google.com
        refresh-token: 18mxeZ5_AE.jkYklrMAf5.IMXnB_DsBY5up4WbYNF2PrY
      name: oidc
```

3. Copy everything after users: and append it to your existing user list in the $HOME/admin.conf. Now you have 2 users: one from the new cluster configuration and one that you added.

### For UI access through OIDC
1. Configure the 'client_iD' and 'authority' in the startup/config yaml with the exact values as ones being used by the Core API Server. Ideally they can be configured through the UI application's Pod manifest using environment variables.

### Setting up RBAC for the new user.
1. Since the new user comes with no permission any access would be by default forbidden
Ex:
```
# kubectl --user=someone@gmail.com get nodes
Error from server (Forbidden): User "someone@gmail.com" cannot list nodes at the cluster scope. (get nodes)
```
2. Grant Permissions: (Make the user part of the admin group)
```
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: admin-role
rules:
  - apiGroups: ["*"]
    resources: ["*"]
    verbs: ["*"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: admin-binding
subjects:
  - kind: User
    name: someone@gmail.com
roleRef:
  kind: ClusterRole
  name: admin-role
  apiGroup: rbac.authorization.k8s.io
```

### Success!
```
# kubectl --user=someone@gmail.com get nodes
NAME      STATUS    AGE       VERSION
node-01   Ready     6h        v1.6.4
```
