# Prometheus Docker Image

This repo builds the [official prometheus](https://github.com/prometheus/prometheus) binaries and repackages them into a hardened container image.

## Building Images

The Prometheus binaries built from the make commands within the prometheus submodule with Prometheus build tool [Promu](https://github.com/prometheus/promu). The customized `Dockerfile` imports necessary dependencies into the image.

```bash
make image
```
