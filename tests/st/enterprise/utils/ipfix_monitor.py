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
import collections
import copy
import logging
import multiprocessing
import netaddr
import os
import Queue
import select
import signal
import subprocess32 as subprocess
import time

from tests.st.utils.utils import debug_failures

devnull = open(os.devnull, 'wb')

_log = logging.getLogger(__name__)
_log.setLevel(logging.DEBUG)

class IpfixFlow(object):

    """

    Represents an IPFIX flow for verification.  Values of None will not be matched against.
    Note that the packets and octets fields take the form "X,Y", where X is packets/octets out, and Y in.

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
         a_octets) = tshark_line.rstrip("\n").split("\t")
        if a_flowset_id == "2":
            # A template, rather than a flow.
            return False
        if self.e_protocol is not None and a_protocol != self.e_protocol:
            return False
        if self.e_srcaddr is not None and a_srcaddr != self.e_srcaddr:
            return False
        if self.e_srcport is not None and a_srcport != self.e_srcport:
            return False
        if self.e_dstaddr is not None and a_dstaddr != self.e_dstaddr:
            return False
        if self.e_dstport is not None and a_dstport != self.e_dstport:
            return False
        if self.e_packets is not None and a_packets != self.e_packets:
            return False
        if self.e_octets is not None and a_octets != self.e_octets:
            return False
        return True

    def __repr__(self):
        return "Flow: " + self.e_srcaddr + " -> " + self.e_dstaddr


class IpfixMonitor(object):
    def __init__(self, collector_addr, collector_port):
        # netcat actually listens on the ipfix port (needed for template emission).
        # Without the template, tshark is unable to decode the data packets.
        # A real collector will always have the socket open anyway.
        self.netcat = subprocess.Popen(["nc", "-ul", collector_addr, str(collector_port)],
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
                    "tshark", "-l", "-i", "any", "-f", "port " + str(collector_port), "-T", "fields",
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

        self.tshark_process = multiprocessing.Process(target=_tshark_loop, args=(self.flows_queue,))
        self.tshark_process.start()

    def __del__(self):
        # It might be nicer to refactor this to use __enter__/__exit__, and invoke using 'with'.
        self.tshark_process.terminate()
        self.netcat.terminate()

    """
    Assert that the collected IPFIX flows are as expected within `timeout` seconds.
    """
    @debug_failures
    def assert_flows_present(self, flows, timeout, allow_others=True):
        flows_left = copy.copy(flows)
        flows_seen = collections.defaultdict(int)

        class TimeoutError(Exception):
            pass

        def handler(signum, frame):
            raise TimeoutError()

        try:
            # Use a signal to give up checking after the timeout.
            signal.signal(signal.SIGALRM, handler)
            signal.alarm(timeout)

            # Look for each expected flow.
            while len(flows_left) > 0 or not allow_others:
                line = self.flows_queue.get(timeout=timeout)
                if line == "2\t\t\t\t\t\t\t\n" or line == "256\t\t\t\t\t\t\t\n":
                    # Skip these lines: the 2... lines are data templates, which tshark needs but don't contain flows.
                    # The 256... lines are data packets received before the template, so tshark couldn't decode them.
                    continue
                flows_seen[line] += 1
                for flow in flows_left:
                    if flow.check_tshark(line):
                        flows_left.remove(flow)
                        break
        except Queue.Empty:
            pass
        except TimeoutError:
            pass
        finally:
            signal.alarm(0)

        # Check we observed all the expected flows, and optionally no others.
        flows_msg = "Expected flows:\r\n %s\r\n" % flows + \
                    "Missing flows:\r\n %s\r\n" % flows_left + \
                    "Observed flows:\r\n %s\r\n" % flows_seen
        assert allow_others or len(flows_seen) <= len(flows), ("Ipfix check error: superfluous flows!\r\n" + flows_msg)
        assert len(flows_left) == 0, ("Ipfix check error: missing flows!\r\n" + flows_msg)

    def reset_flows(self):
        # This isn't entirely deterministic, since flows could be coming in constantly.
        # It should at least clear out all flows that were queued before this call,
        # plus min(10 flows, 1 second) more.
        _log.debug("Resetting flows")
        size = self.flows_queue.qsize()
        while --size > -10:
            try:
                self.flows_queue.get(timeout=0.1)
            except Queue.Empty:
                pass
