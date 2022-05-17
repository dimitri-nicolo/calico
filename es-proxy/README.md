# Elasticsearch Access Proxy

[![Build Status](https://semaphoreci.com/api/v1/projects/8057c105-2db0-41f2-9fa5-d772e81803ac/2294605/badge.svg)](https://semaphoreci.com/calico/es-proxy-image)
[![Docker Image](https://img.shields.io/badge/docker%20image-gcr.io%2Funique--caldron--775%2Fcnx%2Fes--proxy-blue.svg)](https://console.cloud.google.com/gcr/images/unique-caldron-775/GLOBAL/cnx/tigera/es-proxy?project=unique-caldron-775)

Elasticsearch access proxy enables access a password protected Elasticsearch cluster.

## Building the image

Build an image using the `make image` command.

## Example

- Run Elasticsearch (not password protected yet):

  ```bash
  docker stop es; sleep 5; docker run --rm -d -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" --name es docker.elastic.co/elasticsearch/elasticsearch:7.16.2
  ```

- Run the proxy.

  ```bash
  docker run --net=host --rm -it -e LOG_LEVEL=debug -e LISTEN_ADDR=":8080" -e ELASTIC_HOST=localhost -e ELASTIC_PORT=9200 -e ELASTIC_SCHEME=http --name eb-test tigera/es-proxy:latest
  ```

- curl it.

  ```bash
  docker run --net=host --rm byrnedo/alpine-curl -s http://localhost:8080/tigera_secure_ee_audit*/_search -v
  ```
