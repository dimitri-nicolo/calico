# Tigera customized Kibana

Upstream Kibana with Tigera branding. Check `Makefile` and `Dockerfile` for
upstream Kibana version.

Note that unlike other repositories, we don't tag this release with a
Tiger Secure EE release. We instead tag with the corresponding upstream
Kibana version. More on this below.

## Customization

This repository follows similar steps done at [reference repo](https://github.com/Gradiant/dockerized-kibana)

### Updating the Kibana version

After following the steps from the [reference repo](https://github.com/Gradiant/dockerized-kibana)
do the following to update the Kibana version for building images.

* Ensure that the package.json references the correct Kibana version.
* Ensure that the Dockerfile references the correct Kibana version.
* Update the `KIBANA_VERSION` variable in the Makefile.

Images built will be tagged with the value of `KIBANA_VERSION`.

### Building the image

Run `make ci`
