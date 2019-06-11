---
title: Creating kubeconfig files
redirect_from: latest/usage/federation/kubeconfig
canonical_url: https://docs.tigera.io/v2.3/usage/federation/kubeconfig
---

Before installing {{site.prodname}}, you must complete the following steps on each cluster in the federation.

1. Access the cluster using a `kubeconfig` with administrative privileges.

1. If RBAC is enabled, apply the manifest that matches the cluster's datastore type.

   - **Kubernetes API datastore**
     ```bash
     kubectl apply -f \
     {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/federation-rem-rbac-kdd.yaml
     ```

   - **etcd datastore**
     ```bash
     kubectl apply -f \
     {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/federation-rem-rbac-etcd.yaml
     ```

1. Apply the following manifest to create a service account called `tigera-federation-remote-cluster`.

   ```bash
   kubectl apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/federation-remote-sa.yaml
   ```

1. Use the following command to retrieve the name of the secret containing the token associated
   with the `tigera-federation-remote-cluster` service account.

   ```bash
   kubectl describe serviceaccounts tigera-federation-remote-cluster -n kube-system
   ```

   It should return something like the following.

   ```bash
   Name:         tigera-federation-remote-cluster
   Namespace:    kube-system
   Labels:       <none>
   Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"v1","kind":"ServiceAccount","metadata":{"annotations":{},"name":"remote-cluster","namespace":"kube-system"}}

   Image pull secrets:  <none>
   Mountable secrets:   tigera-federation-remote-cluster-token-wzdgp
   Tokens:              tigera-federation-remote-cluster-wzdgp
   Events:              <none>
   ```

   The value of `Tokens` is the name of the secret containing the service account's token.

1. Use the following command to retrieve the token of the service account.

   ```bash
   kubectl describe secrets tigera-federation-remote-cluster-token-wzdgp -n kube-system
   ```

   It should return something like the following.

   ```bash
   Name:         tigera-federation-remote-cluster-token-wzdgp
   Namespace:    kube-system
   Labels:       <none>
   Annotations:  kubernetes.io/service-account.name=tigera-federation-remote-cluster
                 kubernetes.io/service-account.uid=b3044d23-96b5-11e8-ac9b-42010a80000e

   Type:  kubernetes.io/service-account-token

   Data
   ====
   ca.crt:     1025 bytes
   namespace:  11 bytes
   token:      eyJhbGciOiJSUzI1NiIsImtpZCI6IiJ9.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJrdWJlLXN5c3RlbSIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VjcmV0Lm5hbWUiOiJyZW1vdGUtY2x1c3Rlci10b2tlbi13emRncCIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50Lm5hbWUiOiJyZW1vdGUtY2x1c3RlciIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50LnVpZCI6ImIzMDQ0ZDIzLTk2YjUtMTFlOC1hYzliLTQyMDEwYTgwMDAwZSIsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDprdWJlLXN5c3RlbTpyZW1vdGUtY2x1c3RlciJ9.h8ngEwzjzHLMnaRANiXoqrSAWGxycVqq7cO54RM56qyyy_KlAbLpjbhHiaQBNAqJ_LTvjSZ23r2vZn-ZUbTDcoHninD4N2GXKygyVxoBeBzBJinbHWTPp6BYnLvM1pifnj5QNrQanqb0Nwy_p9T1CBMr7NmTsJ5HvRHASCMImjLToCC251kL5oIVM6MWdty_dKGvCzO1rUQhCqcwQyq4Bg6cTFNCLejFpgH0p7XdVcqSsd2uYUpPeS85q5paEKza630Dxg8jdwa5VhYAb_LZfklPOVwHAgNx9OT-z_ZRLYfWoBVlkgazXiiEz9kDweK8hESGLQdW7996C0vdeVx21A
   ```

1. Save the `token` value for later steps.

1. Use the following command to retrieve the certificate authority and server data.

   ```bash
   kubectl config view --flatten --minify
   ```

   It should return something like the following.

   ```bash
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

1. Save the `certificate-authority-data` and the `server` values for the next step.

1. Open a new file in your favorite editor.

   > **Tip**: We recommend coming up with a naming scheme for your clusters that makes sense for your
   > specific use case. For example, if you have four remote clusters that are similar except that they
   > are serve web pages in different languages, you can name the files `kubeconfig-rem-cluster-en`,
   > `kubeconfig-rem-cluster-es`, `kubeconfig-rem-cluster-ja`, and `kubeconfig-rem-cluster-sw`. This will
   > make the installation procedure easier to follow.
   {: .alert .alert-success}

1. Paste the following into your new file.

   ```yaml
   apiVersion: v1
   kind: Config
   users:
   - name: tigera-federation-remote-cluster
     user:
       token: <YOUR-SERVICE-ACCOUNT-TOKEN>
   clusters:
   - name: tigera-federation-remote-cluster
     cluster:
       certificate-authority-data: <YOUR-CERTIFICATE-AUTHORITY-DATA>
       server: <YOUR-SERVER-ADDRESS>
   contexts:
   - name: tigera-federation-remote-cluster-ctx
     context:
       cluster: tigera-federation-remote-cluster
       user: tigera-federation-remote-cluster
   current-context: tigera-federation-remote-cluster-ctx
   ```

1. Replace `<YOUR-SERVICE-ACCOUNT-TOKEN>`, `<YOUR-CERTIFICATE-AUTHORITY-DATA>`,
   and `<YOUR-SERVER-ADDRESS>` with the values obtained in the previous steps.

1. Verify that the `kubeconfig` file works by issuing the following command.

   ```bash
   kubectl --kubeconfig=kubeconfig-rem-cluster-n get services
   ```

   You should see your cluster listed.
