FROM alpine:3.9 as base

# Put our binary in /code rather than directly in /usr/bin.  This allows the downstream builds
# to more easily extract the apiserver build artifacts from the container.
RUN mkdir /code
RUN chgrp -R 0 /code && \
    chmod -R g=u /code
ADD bin/calico-k8sapiserver /code
WORKDIR /code
RUN ln -s /code/calico-k8sapiserver /usr/bin

# Run apiserver by default
ENTRYPOINT ["calico-k8sapiserver"]
