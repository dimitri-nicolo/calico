# Tigera branded Kibana

Upstream Kibana with Tigera branding. Check Kibana submodule for upstream Kibana version.

## Build the image

Run `make ci` to get the final Tigera customized Kibana image. Even though the `Dockerfile` is based on `docker.elastic.co/kibana/kibana`, it is not the official Kibana image. You can run `make build` to get the patched Kibana base image locally and build the final image from it.

## Update version and patch

To update Kibana version, update the `KIBANA_VERSION` variable in the `third_party/kibana/Makefile`.

To update Tigera customization patch, follow the next steps:

1. Clone the upstream [Kibana](https://github.com/elastic/kibana) repository to your develop machine.
2. Switch to a target release tag.
3. Apply all patches under the `patches` folder to your clone and resolve conflicts.
4. Make your changes and verify upstream Kibana still builds after your changes.
5. Use [`git commit`](https://git-scm.com/docs/git-commit) to commit your changes into your clone. It can be multiple commits and  you don't need to push them.
6. Use [`git format-patch`](https://git-scm.com/docs/git-format-patch) to generate patch files. If you have multiple commits, you need to generate one patch file for each commit.
7. Copy patch files back to the `third_party/kibana/patches` folder and update the `init-source` target in `third_party/kibana/Makefile`.
8. Build and validate.

## Customization

This repository follows similar steps done at [reference repo](https://github.com/Gradiant/dockerized-kibana).
