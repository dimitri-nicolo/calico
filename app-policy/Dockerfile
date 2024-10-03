# Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ARG UBI8_IMAGE

FROM ${UBI8_IMAGE} AS builder

# Update base packages to pick up security updates and embedding iptables
RUN microdnf upgrade && microdnf install findutils iptables && \
  ldd /sbin/iptables | tr -s '[:blank:]' '\n' | grep '^/' | \
  xargs -I % sh -c 'mkdir -p $(dirname /deps%); cp % /deps%;'

RUN mkdir -p /var/run/dikastes

FROM scratch AS source

ARG BIN_DIR
ARG TARGETARCH

# Precreate the /var/run/dikastes directory so that we don't need any elevated
# permission to create the directory at runtime.
COPY --from=builder /var/run/dikastes /var/run/dikastes/
COPY --from=builder /sbin/iptables /sbin/iptables
COPY --from=builder /usr/lib64/xtables /usr/lib64/xtables
COPY --from=builder /deps /

COPY ${BIN_DIR}/dikastes-${TARGETARCH} /dikastes

FROM calico/base

ARG GIT_VERSION=unknown

LABEL description="Calico Dikastes enables Application Layer Policy"
LABEL maintainer="maintainers@tigera.io"
LABEL name="Calico Dikastes"
LABEL release="1"
LABEL summary="Calico Dikastes enables Application Layer Policy"
LABEL vendor="Project Calico"
LABEL version=${GIT_VERSION}

COPY --from=source / /

ENTRYPOINT ["/dikastes"]
