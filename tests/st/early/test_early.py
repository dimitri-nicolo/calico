# Copyright (c) 2021 Tigera, Inc. All rights reserved.

import logging
import os
import shutil
import tempfile
import time

from tests.st.test_base import TestBase
from tests.st.utils.docker_host import CHECKOUT_DIR, NODE_CONTAINER_NAME
from tests.st.utils.utils import log_and_run
from tests.k8st.utils.utils import retry_until_success

_log = logging.getLogger(__name__)
_log.setLevel(logging.DEBUG)

class TestEarly(TestBase):

    def setUp(self):
        self.cfgpath = "/code/early_cfg.yaml"
        if os.path.exists(self.cfgpath):
            os.remove(self.cfgpath)

    def tearDown(self):
        log_and_run("docker rm -f calico-early-test", raise_exception_on_failure=False)
        log_and_run("docker rm -f calico-early", raise_exception_on_failure=False)
        log_and_run("docker network rm plane1", raise_exception_on_failure=False)
        log_and_run("docker network rm plane2", raise_exception_on_failure=False)
        if os.path.exists(self.cfgpath):
            os.remove(self.cfgpath)

    def test_early(self):
        """
        Mainline test for early networking.
        """
        log_and_run("docker network create --subnet=10.19.11.0/24 --ip-range=10.19.11.0/24 plane1")
        log_and_run("docker network create --subnet=10.19.12.0/24 --ip-range=10.19.12.0/24 plane2")
        log_and_run("docker create --privileged --name calico-early" +
                    " -v " + CHECKOUT_DIR + ":/code" +
                    " -e CALICO_EARLY_NETWORKING=" + self.cfgpath +
                    " --net=plane1 tigera/cnx-node:latest-amd64")
        log_and_run("docker network connect plane2 calico-early")
        log_and_run("docker start calico-early")

        # Get IP addresses.
        ips = log_and_run("docker inspect '--format={{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}' calico-early").strip().split()
        assert len(ips) == 2, "calico-early container should have two IP addresses"

        # Write early networking config.
        f = open(self.cfgpath, "w")
        f.write("""
kind: EarlyNetworkConfiguration
spec:
  nodes:
    - interfaceAddresses:
        - %s
        - %s
      stableAddress:
        address: 10.19.10.19
      asNumber: 65432
      peerings:
        - peerIP: 10.19.11.1
        - peerIP: 10.19.12.1
""" % (ips[0], ips[1]))
        f.close()
        _log.info("Wrote early networking config to " + self.cfgpath)

        def early_networking_setup_done():
            # Check the container's log.
            logs = log_and_run("docker logs calico-early")
            assert "Early networking set up; now monitoring BIRD" in logs

            # Check that BIRD is running.
            protocols = log_and_run("docker exec calico-early birdcl show protocols")
            assert "tor1" in protocols
            assert "tor2" in protocols
            routes = log_and_run("docker exec calico-early birdcl show route")
            assert "10.19.10.19/32" in routes

        retry_until_success(early_networking_setup_done, retries=3)

        # Create a test container on the same networks.
        log_and_run("docker create --privileged --name calico-early-test" +
                    " --net=plane1 spaster/alpine-sleep")
        log_and_run("docker network connect plane2 calico-early-test")
        log_and_run("docker start calico-early-test")
        retry_until_success(lambda: log_and_run("docker exec calico-early-test ip r"),
                            retries=3)
        log_and_run("docker exec calico-early-test apk update")
        log_and_run("docker exec calico-early-test apk add iproute2 iputils")
        log_and_run("docker exec calico-early-test ip r a 10.19.10.19/32" +
                    " nexthop via " + ips[0] +
                    " nexthop via " + ips[1])
        log_and_run("docker exec calico-early-test ip r")

        # Check that the test container can ping the early container
        # on its stable address.
        pings = log_and_run("docker exec calico-early-test ping -c 7 10.19.10.19")
        assert "7 received, 0% packet loss" in pings
