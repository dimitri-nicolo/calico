ARG UBI_IMAGE

FROM ${UBI_IMAGE} AS ubi

COPY clean.sh /usr/local/bin/clean.sh
RUN touch /in-the-container

# Update base packages to pick up security updates. 
RUN microdnf upgrade && microdnf install curl

RUN curl -OL https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 
RUN chmod +x jq-linux64
RUN mv jq-linux64 /bin/jq

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
COPY bin/scaleloader-amd64 /usr/bin/compliance-scaleloader
COPY playbooks/ /playbooks/
COPY scenarios/ /playbooks/
WORKDIR /playbooks

SHELL ["/bin/bash", "-c"]
RUN  /usr/local/bin/clean.sh "/entrypoint.sh" "/usr/bin/compliance-scaleloader"

FROM scratch
SHELL ["/bin/sh", "-c"]

COPY --from=ubi /entrypoint.sh /entrypoint.sh
COPY --from=ubi /bin/ /bin/
COPY --from=ubi /usr/bin/ /usr/bin/
# copies the shared linux libs requred by compliance-scaleloader identified by ldd bin/scaleloader-amd64`
COPY --from=ubi /lib64/ld-linux-x86-64.so.2 /lib64/ld-linux-x86-64.so.2
COPY --from=ubi /lib64/libc.so.6 /lib64/libc.so.6
COPY --from=ubi /lib64/libtinfo.so.6 /lib64/libtinfo.so.6
COPY --from=ubi /lib64/libdl.so.2 /lib64/libdl.so.2
COPY --from=ubi /lib64/libpthread.so.0 /lib64/libpthread.so.0

ENV RUN_SCENARIO=true

USER 10001:10001

ENTRYPOINT ["/entrypoint.sh"]
