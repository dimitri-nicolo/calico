# Elasticsearch Access Proxy

Elasticsearch access proxy enables access a password protected Elasticsearch cluster.

## Building the image

Build an image using the `make image` command.

## Example

- Run Elasticsearch (not password protected yet):
  ```
  docker stop es; sleep 5; docker run --rm -d -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" --name es docker.elastic.co/elasticsearch/elasticsearch:6.4.1
  ```
- Run the proxy
  ```
  docker run --net=host --rm -it -e TARGET_URL=http://localhost:9200 --name eb-test tigera/es-proxy:latest
  ```

- curl it.
  ```
  docker run --net=host --rm byrnedo/alpine-curl -s http://localhost:8080
  ```
