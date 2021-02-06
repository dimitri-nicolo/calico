## Install {{include.cli}} as a Kubernetes pod


Run the following to deploy the `calicoctl` container to your nodes.

  ```bash
  kubectl apply -f {{ "/manifests/calicoctl.yaml" | absolute_url }}
  ```

  > **Note**: You can also
  > [view the YAML in a new tab]({{ "/manifests/calicoctl.yaml" | absolute_url }}){:target="_blank"}.
  {: .alert .alert-info}

You can then run commands using kubectl as shown below.

```bash
kubectl exec -ti -n kube-system calicoctl -- /calicoctl get profiles -o wide
```

An example response follows.

```
NAME                 TAGS
kns.default          kns.default
kns.kube-system      kns.kube-system
```
{: .no-select-button}

We recommend setting an alias as follows.

```bash
alias calicoctl="kubectl exec -i -n kube-system calicoctl -- /calicoctl"
```

> **Note**: In order to use the `calicoctl` alias
> when reading manifests, redirect the file into stdin, for example:
> ```bash
   > calicoctl create -f - < my_manifest.yaml
   > ```
{: .alert .alert-info}