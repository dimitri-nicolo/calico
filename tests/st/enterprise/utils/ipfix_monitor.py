# Copyright 2016 Tigera, Inc
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import copy
import multiprocessing
import netaddr
import os
import select
import signal
import subprocess32 as subprocess
import time

from tests.st.utils.utils import debug_failures

devnull = open(os.devnull, 'wb')


class IpfixFlow():
    """
    Represents an IPFIX flow for verification.  Values of None will not be matched against.
    """
    def __init__(self, srcaddr, dstaddr, protocol=None, srcport=None, dstport=None, packets=None, octets=None):
        self.e_protocol = protocol
        self.e_srcaddr = srcaddr
        self.e_srcport = srcport
        self.e_dstaddr = dstaddr
        self.e_dstport = dstport
        self.e_packets = packets
        self.e_octets = octets

    def check_tshark(self, tshark_line):
        (a_flowset_id,
         a_protocol,
         a_srcaddr,
         a_srcport,
         a_dstaddr,
         a_dstport,
         a_packets,
         a_octets) = tshark_line.split("\t")
        if a_flowset_id.startswith("FlowSet Id: Data Template"):
            # A template, rather than a flow.
            return False
        # TODO (Matt): Check the other fields as well!
        if self.e_protocol is not None and a_protocol != self.e_protocol:
            return False
        if self.e_srcaddr is not None and a_srcaddr != self.e_srcaddr:
            return False
        if self.e_dstaddr is not None and a_dstaddr != self.e_dstaddr:
            return False
        if self.e_packets is not None and a_packets != self.e_packets:
            return False
        return True


class IpfixMonitor():
    def __init__(self):
        # netcat actually listens on the ipfix port (needed for template emission).
        # Without the template, tshark is unable to decode the data packets.
        # A real collector will always have the socket open anyway.
        self.netcat = subprocess.Popen(["netcat", "-ul", "4739"],
                                       shell=False,
                                       stdout=devnull,
                                       stderr=devnull)
        # Output will be lines with tab separated values for each of the fields set with "-e".
        self.flows_queue = multiprocessing.Queue()

        """
        Run an instance of tshark to capture and decode ipfix data and write the output lines to self.flows_queue.
        Each line contains the fields selected with "-e", tab separated.
        """
        def _tshark_loop(flows_queue):
            tshark = subprocess.Popen(
                [
                    "tshark", "-i", "lo", "-f", "port 4739", "-T", "fields",
                    "-e", "cflow.flowset_id",
                    "-e", "cflow.protocol",
                    "-e", "cflow.srcaddr",
                    "-e", "cflow.srcport",
                    "-e", "cflow.dstaddr",
                    "-e", "cflow.dstport",
                    "-e", "cflow.permanent_packets",
                    "-e", "cflow.permanent_octets",
                ],
                shell=False,
                stdout=subprocess.PIPE,
                stderr=devnull)
            while True:
                line = tshark.stdout.readline()
                # For some reason we get a blank line between each real pair.
                if "\t" in line:
                    flows_queue.put(line)

        self.tshark_process = multiprocessing.Process(target=_tshark_loop, args=self.flows_queue)
        self.tshark_process.start()

    def __del__(self):
        # It might be nicer to refactor this to use __enter__/__exit__, and invoke using 'with'.
        self.tshark_process.terminate()
        self.netcat.terminate()

    """
    Assert that the collected IPFIX flows are as expected within `timeout` seconds.
    """
    # TODO (Matt): Also need to be able to assert no unexpected flows.
    @debug_failures
    def assert_flows_present(self, flows, timeout):
        flows_left = copy.copy(flows)

        class TimeoutError(Exception):
            pass

        def handler(signum, frame):
            raise TimeoutError()

        try:
            # Use a signal to give up checking after the timeout.
            signal.signal(signal.SIGALRM, handler)
            signal.alarm(timeout)

            # Look for each expected flow.
            while len(flows_left) > 0:
                line = self.flows_queue.get(timeout=timeout)
                print line
                for flow in flows_left:
                    if flow.check_tshark(line):
                        flows_left.remove(flow)
                        break
        except multiprocessing.Queue.Empty:
            pass
        except TimeoutError:
            pass
        finally:
            signal.alarm(0)
        assert len(flows_left) == 0, ("Ipfix check error!\r\n" +
                                      "Expected flows:\r\n %s\r\n" % flows +
                                      "Missing flows:\r\n %s\r\n" % flows_left)

    def reset_flows(self):
        # This isn't entirely deterministic, since flows could be coming in constantly.
        # It should at least clear out all flows that were queued before this call,
        # plus min(10 flows, 1 second) more.
        size = self.flows_queue.qsize()
        while --size > -10:
            try:
                self.flows_queue.get(timeout=0.1)
            except multiprocessing.Queue.Empty:
                return

# Test (must be root!!!):
import ipfix_monitor
mon = ipfix_monitor.IpfixMonitor()
fl = ipfix_monitor.IpfixFlow("192.168.123.131", "192.168.123.132")
mon.assert_flows_present([fl])