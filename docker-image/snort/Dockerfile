#
# Dockerfile for snort
#

FROM vimagick/snort

FROM scratch

COPY --from=vimagick/snort /usr/local/lib/ /usr/local/lib/

COPY --from=vimagick/snort /lib64/libc.so.6 /lib64/libc.so.6
COPY --from=vimagick/snort /lib64/libgcc_s.so.1 /lib64/libgcc_s.so.1
COPY --from=vimagick/snort /lib64/libc-2.17.so /lib64/libc-2.17.so
COPY --from=vimagick/snort /lib64/ld-linux-x86-64.so.2 /lib64/ld-linux-x86-64.so.2
COPY --from=vimagick/snort /lib64/libpthread.so.0 /lib64/libpthread.so.0
COPY --from=vimagick/snort /lib64/libz.so.1 /lib64/libz.so.1
COPY --from=vimagick/snort /lib64/libdl.so.2 /lib64/libdl.so.2

COPY --from=vimagick/snort /lib64/libpcre.so.1 /lib64/libpcre.so.1
COPY --from=vimagick/snort /lib64/libnsl.so.1 /lib64/libnsl.so.1
COPY --from=vimagick/snort /lib64/libm.so.6 /lib64/libm.so.6
COPY --from=vimagick/snort /lib64/libcrypto.so.10 /lib64/libcrypto.so.10
COPY --from=vimagick/snort /lib64/libsfbpf.so.0 /lib64/libsfbpf.so.0
COPY --from=vimagick/snort /lib64/libpcap.so.1 /lib64/libpcap.so.1
COPY --from=vimagick/snort /usr/lib64/libdnet.so.1 /usr/lib64/libdnet.so.1
COPY --from=vimagick/snort /usr/lib64/liblzma.so.5 /usr/lib64/liblzma.so.5
COPY --from=vimagick/snort /usr/lib64/liblzma.so.5.2.2 /usr/lib64/liblzma.so.5.2.2

COPY --from=vimagick/snort /usr/sbin/snort /usr/sbin/snort
COPY --from=vimagick/snort /lib64/snort-2.9.16_dynamicengine /lib64/snort-2.9.16_dynamicengine
COPY --from=vimagick/snort /lib64/snort-2.9.16_dynamicpreprocessor /lib64/snort-2.9.16_dynamicpreprocessor

COPY --from=vimagick/snort /usr/lib64/snort-2.9.16_dynamicpreprocessor/ /usr/lib64/snort-2.9.16_dynamicpreprocessor/
COPY --from=vimagick/snort /usr/lib64/snort-2.9.16_dynamicengine/ /usr/lib64/snort-2.9.16_dynamicengine/

COPY --from=vimagick/snort /etc/snort/ /etc/snort/
COPY --from=vimagick/snort /var/log/snort/ /var/log/snort/


COPY snort.conf /etc/snort/snort.conf
COPY local.rules /etc/snort/rules/local.rules

ENTRYPOINT ["snort"]
CMD ["--help"]

