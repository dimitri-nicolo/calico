---
title: Writing kubeconfig files to connect to your Kubernetes cluster
---

Instead of connecting to your Kubernetes cluster using the corresponding admin kubeconfig
file (usually created automatically for you and found at ~/.kube/config), we recommend
creating a custom kubeconfig file which allows for more controlled access to your
Kubernetes cluster. This document will explain the steps required in order to properly
write kubeconfig files and configure appropriate access permissions.

## Overview
We recommend accessing Kubernetes clusters with an isolated set of credentials
by creating a [Kubernetes Service Account](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/){:target="_blank"}
which will manage access permissions and writing a kubeconfig file that can utilize it.

### Creating a service account
In order to create permissions to a remote Kubernetes cluster, we must already have
read and write access to it. Using your admin kubeconfig file (or other kubeconfig file
with read and write permissions), create a service account that looks similar to the following.

```
apiVersion: v1
kind: ServiceAccount
metadata:
  name: remote-cluster
  namespace: kube-system
```

Once the service account has been created, it is important to define the appropriate RBAC
permissions. An example of what those RBAC permissions may look like is provided below.

```
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: remote-cluster
rules:
  - apiGroups: [""]
    resources:
      - namespaces
      - serviceaccounts
      - services
      - endpoints
    verbs:
      - get
      - list
      - watch
  - apiGroups: [""]
    resources:
      - pods
    verbs:
      - get
      - list
      - watch
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - hostendpoints
      - profiles
    verbs:
      - create
      - get
      - list
      - update
      - watch

---

apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: remote-cluster
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: remote-cluster
subjects:
- kind: ServiceAccount
  name: remote-cluster
  namespace: kube-system
```

Once you have applied the service account and its RBAC permissions to your cluster, you
are ready to write a kubeconfig file to utilize it.

### Writing a kubeconfig file
Before we create the kubeconfig file, we need to collect information about the
cluster and the service account. In order to accomplish this, we will need access to
an existing kubeconfig file for our cluster as well as the ability to access the
service account information that we are configuring our kubeconfig file to inherit
permissions from.

#### Service account token
The first piece of information that we require is the secret token associated with the account.
Use the following command to get this information.

```
kubectl describe serviceaccounts remote-cluster -n kube-system
```

You should see output similar to the following.

```
Name:         remote-cluster
Namespace:    kube-system
Labels:       <none>
Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"v1","kind":"ServiceAccount","metadata":{"annotations":{},"name":"remote-cluster","namespace":"kube-system"}}

Image pull secrets:  <none>
Mountable secrets:   remote-cluster-token-wzdgp
Tokens:              remote-cluster-token-wzdgp
Events:              <none>
```

Use the value in the `Tokens` field in order to get the token value.

```
kubectl describe secrets remote-cluster-token-wzdgp -n kube-system
```

This should output something similar to the following.

```
Name:         remote-cluster-token-wzdgp
Namespace:    kube-system
Labels:       <none>
Annotations:  kubernetes.io/service-account.name=remote-cluster
              kubernetes.io/service-account.uid=b3044d23-96b5-11e8-ac9b-42010a80000e

Type:  kubernetes.io/service-account-token

Data
====
ca.crt:     1025 bytes
namespace:  11 bytes
token:      eyJhbGciOiJSUzI1NiIsImtpZCI6IiJ9.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJrdWJlLXN5c3RlbSIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VjcmV0Lm5hbWUiOiJyZW1vdGUtY2x1c3Rlci10b2tlbi13emRncCIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50Lm5hbWUiOiJyZW1vdGUtY2x1c3RlciIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50LnVpZCI6ImIzMDQ0ZDIzLTk2YjUtMTFlOC1hYzliLTQyMDEwYTgwMDAwZSIsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDprdWJlLXN5c3RlbTpyZW1vdGUtY2x1c3RlciJ9.h8ngEwzjzHLMnaRANiXoqrSAWGxycVqq7cO54RM56qyyy_KlAbLpjbhHiaQBNAqJ_LTvjSZ23r2vZn-ZUbTDcoHninD4N2GXKygyVxoBeBzBJinbHWTPp6BYnLvM1pifnj5QNrQanqb0Nwy_p9T1CBMr7NmTsJ5HvRHASCMImjLToCC251kL5oIVM6MWdty_dKGvCzO1rUQhCqcwQyq4Bg6cTFNCLejFpgH0p7XdVcqSsd2uYUpPeS85q5paEKza630Dxg8jdwa5VhYAb_LZfklPOVwHAgNx9OT-z_ZRLYfWoBVlkgazXiiEz9kDweK8hESGLQdW7996C0vdeVx21A
```

Save the token value for later steps.

#### Cluster certificate authority and server address
We will also need to grab the certificate authority information and the server address of
the kubernetes cluster. We can find this information in the `certificate-authority-data`
and `server` fields of the kubeconfig or the output of the following command.

```
kubectl config view --flatten --minify
```

The output should look similar to the following.

```
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN5RENDQWJDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRFNE1EY3lOVEUyTURjek4xb1hEVEk0TURjeU1qRTJNRGN6TjFvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTU9FCjc1ZWtxbmdMdjZKZW9IaGhFLy9iVU0rdnNRMyt3cUZ1eHp5NDlReFk1cmtZaGJ4WWt1TFl0c3ZBVTgycjVnNlYKbUVwbDRlOHJMQjRnTnFpRnVDcVplcmMwWWxqZDVQNllKcW5PN3A1b1J3SG5BWXA3dnZwczJRZjFTemNsTVNjSAo3Qk9XbkF4aEFFZTUzKytPSkhaVmZzL2tlTWtlK3F4b01IZ0R1eGxUb2xXU2dDV0YvbCt3eFk3ZTcyNmozYWZnClRXL0lxSk9GNTN6YTVHR05MMGRIQndFR0gzSU5CVllXa1V5TjEycXZJYlh3RzJReElCU1hEbTcvV1dvNUphMHkKeVNSWkx4L3FZSHBld1hsOU5Od3ZYaXNOZ2xvVVVsZFJ4OGtSWmJOdmViZkpad2FqQVBlRytNRkYwYVJVNHF0Zgo5NVYzdHZ5RHYxaHNJdytGQiswQ0F3RUFBYU1qTUNFd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dFQkFFaFJBNmEzUUNBRGVGalNOeFM1Qi9NZVo4dTIKenNvV2F0c21pdm81YUREUUIwUnRUUWFLTUEwU2ZlaGNLd3ZUdTJjVW5CaVJHZlZMSnJ3REwvUzUvRHBhemdBbgozVlZuWERlbGRzcjVKa3dpTElhQUZCSzg5K3BaLzVybXpuZmZpMldKS0JvY2t3N015REplb05FdklkSjVtb2t0ClMxL0pKYlcvbExZU054RjYxOXJxOE9LVVZ2YStwTmczZ2JEbGlFSUZNUmpobWdtU01tUEthRnMvaE1GcDlzdVMKV3VDd1czY2lQVXlUZXV6bnRYbzY5K3NpUGYyLzFxYUFmRWtmSHp3NS9QeG8xM1dOWnEzSzI0dDNoNXh1QjFvRwpwaXRDbEZPSGFLSnNLZ0g1UkFOSWt3dkNIOXgySG9oTzR2M3ZoOXd1KzlOMEt1K3FLVWJIODRENkVpMD0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
    server: https://10.128.0.14:6443
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: kubernetes-admin
  name: kubernetes-admin@kubernetes
current-context: kubernetes-admin@kubernetes
kind: Config
preferences: {}
users:
- name: kubernetes-admin
  user:
    client-certificate-data: ...
    client-key-data: ...
```

Save the `certificate-authority-data` and the `server` values for the next step.

#### Creating the kubeconfig file
Once we have collected all of the relevant data, we are now ready to create our kubeconfig file.
Write the following to a file replacing the appropriate fields.

```
apiVersion: v1
kind: Config
users:
- name: remote-cluster
  user:
    token: <YOUR SERVICE ACCOUNT TOKEN>
clusters:
- name: remote-cluster
  cluster:
    certificate-authority-data: <YOUR CERTIFICATE AUTHORITY DATA>
    server: <YOUR SERVER ADDRESS>
contexts:
- name: remote-cluster-ctx
  context:
    cluster: remote-cluster
    user: remote-cluster
current-context: remote-cluster-ctx
```

Now that we have a kubeconfig file, it is time to test it. 

```
kubectl --kubeconfig=remote-cluster.cfg get namespaces
```

You should see for your cluster listed.

#### Troubleshooting
In case you are having issues accessing your cluster resources, make
sure to check the following:

1. The `name` field in the `users` section and in the `context` portion of the `contexts` section match
your service account name

1. The `current-context` field matches the `context` with your service account user.

1. You have set the correct RBAC permissions for the resources you are attempting to access.
