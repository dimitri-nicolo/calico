# Prometheus Operator and Config Reloader Container Images

This repo wraps the prometheus-operator and config-reloader binaries so that dependencies can be injected when creating the images. Downloads [prometheus-operator](https://github.com/prometheus-operator/prometheus-operator) source.

## Building Images

Binaries are built from the make commands from the prometheus-operator source.  The customized `Dockerfile`s imports the binary into the images created.

```bash
make image
```

Builds both `tigera/prometheus-operator` and `tigera/prometheus-config-reloader`

```bash
make tigera/prometheus-operator
```

Solely builds the `tigera/prometheus-operator` image

```bash
make tigera/prometheus-config-reloader
```

Solely builds the `tigera/prometheus-config-reloader` image
