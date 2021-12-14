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
6. Under Application type, select Web Application. Set the desired "Authorized JavaScript origins" and "Authorized redirect URIs". The authorized URI will be appended with "id_token"/"access_toke" based on the "response_type". 

For detailed workflow please refer: https://www.ibm.com/support/knowledgecenter/en/SSEQTP_8.5.5/com.ibm.websphere.wlp.doc/ae/twlp_oidc_auth_endpoint.html

7. From the resulting OAuth client dialog box, copy the Client ID. The Client ID lets your app access enabled Google APIs.
8. Download the client secret JSON file of the credentials.

### Setting up a Kubernetes cluster
1. Update the kube api server arguments in the manifest /etc/kubernetes/manifests/kube-apiserver.yaml to set 'oidc-issuer-url', 'oidc-username-claim', 'oidc-client-id'. Following command can be used to perform the action: 

Set the Client ID based on the credential created above.
```
sed -i "/- apiserver/a\    - --oidc-issuer-url=https://accounts.google.com\n    - --oidc-username-claim=email\n    - --oidc-client-id=<FILL_IN_THE_CLIENT_ID_FROM_CONSOLE_PROJECT>" /etc/kubernetes/manifests/kube-apiserver.yaml
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
1. Configure the 'client_iD' and/or 'client_secret' and 'endpoint/authority' in the startup/config yaml with the same values as ones being used to configure the Core API Server. In the workflow so far those values would be the ones that were created as part of the console project/credentials workflow. 

These values should ideally be configured through the UI application's Pod manifest using environment variables.
2. Once the id_token is fetched from the OAuth2 browser workflow, i.e. from the redirect uri query parameters, it can be used to set the Authorization request header. For ex, id_token being embedded into the Authorization header: 
```
curl -L H "Authorization: Bearer eyJhbGREDACTEDlOTE0ZGRkOWY4MGYyOGY2YWU0ZDBhNGMzZTAxOTE1NzFkNTIifQ.eyJhenREDACTEDlobnJudGZsZm5sNW80bGliYm9ibHFmbGkwODQ3NXIuYXBwcy5nb29nbGV1c2VyY29udGVudC5jb20iLCJhdWQiOiI1OREDACTED0bGliYm9ibHFmbGkwODQ3NXIuYXBwcy5nb29nbGV1c2VyY29udGVudC5jb20iLCJzdWIiOiIxMDkwNDY2NDM4MDYzNDk5MTM2NzEiLCJlbWFpbCI6InNREDACTEDUBnbWFpbC5jb20iLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwiYXRfaGFREDACTED3czM2Uk8zSUg5RmpTeHciLCJpc3MiOiJodHRwczovL2FjY291bnRzLmdvb2dsZS5jb20iLCJpYXQiOjE1MDU3MTIwOTUsImV4cCI6MTUwNTREDACTEDRydSBTYWRodSIsInBpY3R1cmUiOiJodHRwczovL2xoNS5nb29nbGV1c2VyY29udGVudC5jb20vLXpMcUxYT19rU3JNL0FBQUFBQUFBQUFJL0FBQUFBQUFBQUFBL0FQSnlwQTNFeDVQbUoyZnpVSGptMVpaYTUteFlRTFdzakEvczk2LWMvcGhvdG8uanBnIiwiZ2l2ZW5fbmFtZSI6IlNoYXRydSIsImZhbWlseV9uYW1lIjoiU2FkaHUiLCJsb2NhbGUiOiJlbiJ9.gAFVQc-6R29VtcrNXA3UTO0U2svu4qwY-LZEMIMPcp7tHg4LnmaXAdab5FTuWKgkG4AXeuXtkNYKJNiYf7yCt_TX90QDV6kUDGPKOWQueX1Gst2jgnCyATHy5hkEvun1XoVC2eFndlsAonIhicAlkO7z86E6bjog5gGfy2M36QgABBOmHyTsO9ueLP1_0yCCgjoQPcK5o4u-VZpIE9G-0I03SaZ664dppHIH1j1GYEaeW1n4xzNVX_Yw6x8qCMbH6QsNkVDTEPh0-y7hGsfzfbk8T-vDVSFriGA3_LiABUADy5WJduwVM5PYWqDtnxCwfYxVo43vWA6O-FRcRf2M3w" --cacert /etc/kubernetes/pki/apiserver.crt https://10.0.2.15:6443/apis/calico.tigera.io/v1/policies

{
  "kind": "PolicyList",
  "apiVersion": "calico.tigera.io/v1",
  "metadata": {
    "selfLink": "/apis/calico.tigera.io/v1/policies",
    "resourceVersion": "60"
  },
  "items": []
}
```

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
