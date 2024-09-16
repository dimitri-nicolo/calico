# Copyright (c) 2018-2024 Tigera, Inc. All rights reserved.
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

FROM alpine:3 AS builder

RUN mkdir -p /usr/bin/l7admctrl

FROM scratch AS source

ARG BIN_DIR
ARG TARGETARCH

# Precreate the /usr/bin/l7admctrl directory so that we don't need any elevated
# permission to create the directory at runtime.
COPY ${BIN_DIR}/l7admctrl-${TARGETARCH} /usr/bin/l7admctrl

FROM calico/base

ARG GIT_VERSION=unknown

LABEL description="Calico L7AdmissionController enables mutating webhooks for Application Layer Policy sidecar injection"
LABEL maintainer="maintainers@tigera.io"
LABEL name="Calico L7AdmissionController"
LABEL release="1"
LABEL summary="Calico L7AdmissionController enables mutating webhooks for Application Layer Policy sidecar injection"
LABEL vendor="Project Calico"
LABEL version=${GIT_VERSION}

COPY --from=source / /

ENTRYPOINT ["/usr/bin/l7admctrl"]
