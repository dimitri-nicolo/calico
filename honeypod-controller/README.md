# Honeypod Controller 
Building on top of Honeypods, a controller is added to monitor traffic into Honeypods. Leveraging our Packet Capture capability, the controller will monitor for Honeypod related alerts and trigger an IDS scan on the packets sent to the Honeypods. If any traffic matches an IDS signature, an alert is created and sent to elasticsearch.

The current implementation contains a DaemonSet deployment that polls the events index for any Honeypod related alerts. If found the controller will determine the correct pod and look for packet capture. If found, snort is used to scan the pcap and any alerts found will be sent to elastic.

## Prerequisite
1. Calico Enterprise 3.3 and above
2. Honeypod manifests are deployed

## Installation
1. `kubectl apply -f install/roles.yaml`
2. `kubectl apply -f install/capture-honey.yaml`
3. `kubectl apply -f install/controller.yaml`

## Testing
`make ut` and e2e has been created.

## Updating Snort2 version

To update the Snort2 version used, update the version number assigned to the variable `SNORT_VERSION` in Makefile.
