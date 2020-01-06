
## Customizing application layer policy manifests

### About customizing application layer policy manifests

Instead of installing from our pre-modified Istio manifests, you may wish to
customize your Istio install or use a different Istio version.  This section
walks you through the necessary changes to a generic Istio install manifest to
allow application layer policy to operate.

### Sidecar injector

The standard Istio manifests for the sidecar injector include a ConfigMap that
contains the template used when adding pods to the cluster. The template adds an
init container and the Envoy sidecar.  Application layer policy requires
an additional lightweight sidecar called Dikastes which receives {{site.tseeprodname}} policy
from Felix and applies it to incoming connections and requests.

If you haven't already done so, download an
[Istio release](https://github.com/istio/istio/releases) and untar it to a
working directory.

Open the `install/kubernetes/istio-demo-auth.yaml` file in an
editor, and locate the `istio-sidecar-injector` ConfigMap.  In the existing `istio-proxy` container, add a new `volumeMount`.

```
        - mountPath: /var/run/dikastes
          name: dikastes-sock
```

Add a new container to the template.

```
      - name: dikastes
        image: {{page.registry}}{{site.imageNames["dikastes"]}}:{{site.data.versions[page.version].first.components["dikastes"].version}}
        args: ["/dikastes", "server", "-l", "/var/run/dikastes/dikastes.sock", "-d", "/var/run/felix/nodeagent/socket", "--debug"]
        volumeMounts:
        - mountPath: /var/run/dikastes
          name: dikastes-sock
        - mountPath: /var/run/felix
          name: felix-sync
```

Add two new volumes.

```
      - name: dikastes-sock
        emptyDir:
          medium: Memory
      - name: felix-sync
        flexVolume:
          driver: nodeagent/uds
```

The volumes you added are used to create Unix domain sockets that allow
communication between Envoy and Dikastes and between Dikastes and
Felix.  Once created, a Unix domain socket is an in-memory communications
channel. The volumes are not used for any kind of stateful storage on disk.

Refer to the
[{{site.tseeprodname}} ConfigMap manifest](/{{page.version}}/manifests/app-layer-policy/istio-inject-configmap.yaml){:target="_blank"} for an
example with the above changes.
