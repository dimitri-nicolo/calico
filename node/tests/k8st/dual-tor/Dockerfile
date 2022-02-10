ARG K8S_VERSION
FROM kindest/node:${K8S_VERSION}

RUN apt-get update
RUN apt-get install -y podman

COPY get-ips.sh /
RUN chmod +x /get-ips.sh

COPY tmp/cnx-node.tar /
RUN mkdir /calico-early
COPY cfg.yaml /calico-early/
