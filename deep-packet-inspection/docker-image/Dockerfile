# Copyright (c) 2023 Tigera, Inc. All rights reserved.

ARG UBI_IMAGE

FROM rockylinux:8 AS snort

ARG SNORT3_VERSION=3.1.76.0
ARG DAQ_VERSION=3.0.13

RUN dnf upgrade -y && dnf install -y \
    autoconf \
    automake \
    bison \
    cmake \
    epel-release \
    flex \
    gcc \
    gcc-c++ \
    libtool \
    pkg-config

RUN dnf --enablerepo=powertools install -y \
    hwloc-devel \
    libdnet-devel \
    libnetfilter_queue-devel \
    libnfnetlink-devel \
    libpcap-devel \
    libuuid-devel \
    luajit-devel \
    openssl-devel \
    pcre-devel \
    xz-devel \
    zlib-devel

RUN mkdir /build
WORKDIR /build

# Install snort DAQ (Data Acquisition library)
RUN curl -sfL https://github.com/snort3/libdaq/archive/refs/tags/v${DAQ_VERSION}.tar.gz | tar xz -C /build && \
    cd /build/libdaq-${DAQ_VERSION}/ && \
    ./bootstrap && ./configure --prefix=/usr --libdir=/usr/lib64 && \
    make -j4 && make install

# Build and Install Snort 3
RUN curl -sfL https://github.com/snort3/snort3/archive/refs/tags/${SNORT3_VERSION}.tar.gz | tar xz -C /build && \
    cd /build/snort3-${SNORT3_VERSION}/ && \
    ./configure_cmake.sh --prefix=/usr --build-type=Release --disable-gdb --disable-docs && \
    cd build && \
    make -j4 && make install

# dry-run the snort config to check for errors
RUN snort -V && snort -c /usr/etc/snort/snort.lua -T

# Create Snorts Log directory
RUN mkdir /var/log/snort /var/log/snort-alerts

FROM scratch as source

ARG TARGETARCH

# Copy snort and dependencies
COPY --from=snort /usr/bin/snort /usr/bin/snort
COPY --from=snort /usr/lib64/libdaq.so.3 /lib64/libdaq.so.3

COPY --from=snort /lib64/libcrypto.so.1.1 /lib64/libcrypto.so.1.1
COPY --from=snort /lib64/libdl.so.2 /lib64/libdl.so.2
COPY --from=snort /lib64/libdnet.so.1 /lib64/libdnet.so.1
COPY --from=snort /lib64/libgcc_s.so.1 /lib64/libgcc_s.so.1
COPY --from=snort /lib64/libhwloc.so.15 /lib64/libhwloc.so.15
COPY --from=snort /lib64/libibverbs.so.1 /lib64/libibverbs.so.1
COPY --from=snort /lib64/libluajit-5.1.so.2 /lib64/libluajit-5.1.so.2
COPY --from=snort /lib64/liblzma.so.5 /lib64/liblzma.so.5
COPY --from=snort /lib64/libm.so.6 /lib64/libm.so.6
COPY --from=snort /lib64/libnl-3.so.200 /lib64/libnl-3.so.200
COPY --from=snort /lib64/libnl-route-3.so.200 /lib64/libnl-route-3.so.200
COPY --from=snort /lib64/libpcap.so.1 /lib64/libpcap.so.1
COPY --from=snort /lib64/libpcre.so.1 /lib64/libpcre.so.1
COPY --from=snort /lib64/libstdc++.so.6 /lib64/libstdc++.so.6
COPY --from=snort /lib64/libuuid.so.1 /lib64/libuuid.so.1
COPY --from=snort /lib64/libz.so.1 /lib64/libz.so.1

# Copy snort scripts and community rules
COPY --from=snort /usr/etc/snort /usr/etc/snort/
COPY docker-image/snort3-community.rules /usr/etc/snort/snort3-community.rules
 
# Copy log location
COPY --from=snort /var/log/snort /var/log/snort/
COPY --from=snort /var/log/snort-alerts /var/log/snort-alerts/

COPY bin/deep-packet-inspection-${TARGETARCH} /usr/bin/deep-packet-inspection

FROM calico/base

COPY --from=source / /

ENTRYPOINT ["/usr/bin/deep-packet-inspection"]
