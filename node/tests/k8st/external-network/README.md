
# External network testing

Scripts in this directory model a KIND cluster setup which includes a cluster node peering with four routers in three external networks.

To run the test manually on your local VM, following the steps below:
- make kind-k8st-setup
- make external-network-setup
- make external-network-run-test or running your own test code
- make external-network-cleanup
- make kind-k8st-cleanup
