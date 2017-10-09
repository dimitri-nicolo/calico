FROM fedora

# Put out binary in /code rather than directly in /usr/bin.  This allows the downstream builds
# to more easily extract the apiserver build artefacts from the container.
RUN mkdir /code
ADD bin/calico-k8sapiserver /code
WORKDIR /code
RUN ln -s /code/calico-k8sapiserver /usr/bin

# Run apiserver by default
ENTRYPOINT ["calico-k8sapiserver"]