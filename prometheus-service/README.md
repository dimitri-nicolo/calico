# Prometheus Service

This directory contains the source for the tigera/prometheus-service image, which acts as a sidecar for the Prometheus pod serving as a proxy that extends apriori functionalities before requests are received by Prometheus. 

Currently it has the following functionalities:
- RBAC check on the user token sent along the request to prometheus