# Copyright (c) 2024 Tigera, Inc. All rights reserved.

FROM rockylinux:8 AS snort

ARG SNORT_VERSION

RUN dnf -y upgrade && dnf -y install \
    epel-release \
    libdnet

RUN dnf -y install https://www.snort.org/downloads/snort/snort-${SNORT_VERSION}-1.centos.x86_64.rpm

RUN mkdir -p /etc/snort/rules \
    /etc/snort/so_rules \
    /etc/snort/preproc_rules \
    /usr/local/lib/snort_dynamicrules

RUN touch /etc/snort/rules/local.rules \
    /etc/snort/rules/black_list.rules \
    /etc/snort/rules/white_list.rules \
    /etc/snort/rules/custom.rules

RUN ln -s /usr/lib64/libdnet.so.1 /usr/lib64/libdnet.1

RUN mv /usr/lib64/snort-${SNORT_VERSION}_dynamicengine/ /usr/lib64/snort_dynamicengine/
RUN mv /usr/lib64/snort-${SNORT_VERSION}_dynamicpreprocessor/ /usr/lib64/snort_dynamicpreprocessor/

COPY community.rules /etc/snort/rules/community.rules
COPY snort.conf /etc/snort/snort.conf

RUN snort -V
