# Copyright (c) 2025 Tigera, Inc. All rights reserved.

FROM scratch

ARG TARGETARCH
COPY ./bin/oiler-${TARGETARCH} /oiler

CMD ["/oiler"]
