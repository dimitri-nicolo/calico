# Alertmanager Container Image

This repo wraps the alertmanager binary so that dependencies can be injected when creating the images. Downloads [alertmanager](https://github.com/prometheus/alertmanager) source.

## Building Images

The Alertmanager binary built from source with Prometheus build tool [Promu](https://github.com/prometheus/promu). The customized `Dockerfile`s imports the binary into the images created.

```bash
make image
```

Builds `tigera/alertmanager` image.
