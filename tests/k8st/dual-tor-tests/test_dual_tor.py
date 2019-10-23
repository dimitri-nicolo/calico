# Copyright (c) 2019 Tigera, Inc. All rights reserved.

import os
import re
import subprocess
import sys
import shlex
import threading
import time
import logging

from kubernetes import client, config

from tests.k8st.test_base import TestBase
from tests.k8st.utils.utils import retry_until_success, DiagsCollector, kubectl

_log = logging.getLogger(__name__)

def output_reader(proc, content):
    while True:
        line = proc.stdout.readline()
        if line:
            content.append(line)
        else:
            break

def run_daemon_with_log(cmd, logs):
    proc1 = subprocess.Popen(shlex.split(cmd), stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    t1 = threading.Thread(target=output_reader, args=(proc1, logs))
    t1.start()

class FailoverTestConfig(object):
    def __init__(self, total_packets, spec_name, max_errors):
        self.total_packets = total_packets
        self.spec_name = spec_name
        self.max_errors = max_errors
        self.timeout = total_packets/100 + 10 # expect receiving 100 packets per seconds plus 10 seconds buffer. 

        # get pod ip and target port
        _, self.client_ip = self.get_pod_ip("pod-name", "client")
        _, self.client_host_ip = self.get_pod_ip("pod-name", "client-host")
        _, self.ra_server_ip = self.get_pod_ip("pod-name", "ra-server")
        _, self.rb_server_ip = self.get_pod_ip("pod-name", "rb-server")
        self.target_port = "8090"

        # get service ip and port
        self.ra_service_ip = self.get_service_ip("ra-server")
        self.rb_service_ip = self.get_service_ip("rb-server")
        self.service_port = "8090"

        # get node ip and port
        self.ra_node_port = self.get_node_port("ra-server")
        self.rb_node_port = self.get_node_port("rb-server")
        self.node_port_ip = "172.31.20.3" # Node loopback address for kind-worker2.

        if spec_name == "host access":
            self.get_plane_from_host()
        else:    
            self.get_plane_from_pod()

        self.output()
   
    def output(self):
        _log.info("Pod IPs: client<%s> client-host<%s> ra-server<%s> rb-server<%s>  port <%s>", self.client_ip, self.client_host_ip, self.ra_server_ip, self.rb_server_ip, self.target_port)
        _log.info("Service IP: ra-server <%s> rb-server <%s> port <%s>", self.ra_service_ip, self.rb_service_ip, self.service_port)
        _log.info("Node IP: <%s> ra-server port <%s> rb-server port <%s>", self.node_port_ip, self.ra_node_port, self.rb_node_port)
        _log.info("Network: rack A   client <%s, %s>  <---> ra-server <%s, %s> ", self.client_ra_plane, self.client_ra_dev, self.ra_server_plane, self.ra_server_dev)
        _log.info("         rack B   client <%s, %s>  <---> rb-server <%s, %s> ", self.client_rb_plane, self.client_rb_dev, self.rb_server_plane, self.rb_server_dev)

    def get_pod_ip(self, key, value):
        cmd="kubectl get pods -n dualtor --selector=\"" + key + "=" + value + "\"" + " -o json 2> /dev/null " + " | jq -r '.items[] | \"\(.metadata.name) \(.status.podIP)\"'"
        output=subprocess.check_output(cmd, shell=True, stderr=subprocess.STDOUT)
        s=output.split()
        if len(s) != 2:
            _log.exception("failed to get pod ip for label %s==%s", key, value)
            raise Exception("error getting pod ip")
        return s[0], s[1]
    
    def get_service_ip(self, name):
        cmd = "kubectl get service " + name + " -n dualtor -o json | jq -r '.spec.clusterIP'"
        output=subprocess.check_output(cmd, shell=True, stderr=subprocess.STDOUT)
        return output.strip()

    def get_node_port(self, name):
        cmd="kubectl get service " + name + " -n dualtor -o json 2> /dev/null | jq -r '.spec.ports[] | \"\(.nodePort)\"'"
        output=subprocess.check_output(cmd, shell=True, stderr=subprocess.STDOUT)
        return output.strip()

    def get_dev(self, output):
        # output should only be 1 or 2
        dev = ""
        if output == "1":
            dev = "eth0"
        elif output == "2":
            dev = "eth1"
        else:
            _log.exception("failed to get interface name from plane output %s", output)
            raise Exception("error getting dev")
    
        return output, dev
    
    def get_plane(self, src_pod_name, dst_ip, route_order):
        # traceroute src --> dst and get the route with route_order
        cmd="kubectl exec -t " + src_pod_name + " -n dualtor -- timeout 5s traceroute -n " + dst_ip + " | grep '" + str(route_order) + "  172'"
        print cmd
        try: 
            output=subprocess.check_output(cmd, shell=True, stderr=subprocess.STDOUT)
        except Exception:
            print "For some reason, running ServiceIP failover test (without tor2tor, tor2node), traceroute could get into the the state that it lost packets on last test case. Print log for now and retry with longer timeout. We will debug it later."
            cmd="kubectl exec -t " + src_pod_name + " -n dualtor -- timeout 25s traceroute -n " + dst_ip + " | grep '" + str(route_order) + "  172'"
            output=subprocess.check_output(cmd, shell=True, stderr=subprocess.STDOUT)

        print output

        m = re.search('172\.31\..(\d)', output)
        if m:
            found = m.group(1)
            return self.get_dev(found)
        else:
            print "something wrong"
            cmd="kubectl exec -t " + src_pod_name + " -n dualtor -- timeout 5s traceroute -n " + dst_ip
            print cmd
            output=subprocess.check_output(cmd, shell=True, stderr=subprocess.STDOUT)
            print output

            _log.exception("failed to match route info to plane. output %s", output)
            raise Exception("error match route info")

    def get_plane_from_pod(self):
        self.client_ra_plane, self.client_ra_dev = self.get_plane("client", self.ra_server_ip, 2)
        self.ra_server_plane, self.ra_server_dev = self.get_plane("ra-server", self.client_ip, 2)
        self.client_rb_plane, self.client_rb_dev = self.get_plane("client", self.rb_server_ip, 2)
        self.rb_server_plane, self.rb_server_dev = self.get_plane("rb-server", self.client_ip, 2)

    def get_plane_from_host(self):
        self.client_ra_plane, self.client_ra_dev = self.get_plane("client-host", self.ra_server_ip, 1)

        # Hosts in same subent, these values are not used.
        self.ra_server_plane = "n/a"
        self.ra_server_dev = "n/a"

        self.client_rb_plane, self.client_rb_dev = self.get_plane("client-host", self.rb_server_ip, 1)
        self.rb_server_plane, self.rb_server_dev = self.get_plane("rb-server", self.client_host_ip, 2)
    
class FailoverTest(object):
    def __init__(self, config):
        self.config = config

        self.ra_error = 0
        self.rb_error = 0

        self.client_func = None
        if self.config.spec_name == "pod ip" or self.config.spec_name == "host access":
            self.client_func = self.client_func_pod_ip

        if self.config.spec_name == "service ip":
            self.client_func = self.client_func_service_ip

        if self.config.spec_name == "node port":
            self.client_func = self.client_func_node_port
         
        if self.client_func == None:
            _log.exception("unknown test spec <%s> specified.", self.config.spec_name)
            raise Exception("unknown test spec")

    def start_server(self, name, server_logs):
        run_daemon_with_log("kubectl exec -n dualtor " + name + " -- nc -l -p " + self.config.target_port, server_logs)
    
    def start_client(self, name, ip, port):
        script="for i in `seq 1 " + str(self.config.total_packets) + "`; do echo $i -- " + name + "; sleep 0.01; done | nc -w 1 " + ip +  " " + port
        if self.config.spec_name == "host access":
            cmd = "kubectl exec -n dualtor -t client -- /bin/sh -c \"" + script + "\""
        else:    
            cmd = "kubectl exec -n dualtor -t client-host -- /bin/sh -c \"" + script + "\""
        proc1 = subprocess.Popen(shlex.split(cmd), stdout=subprocess.PIPE, stderr=subprocess.PIPE)
   
    def print_diag(self, name):
        cmd="docker exec -t kind-worker3 ip r"
        server_output=subprocess.check_output(cmd, shell=True, stderr=subprocess.STDOUT)
        print server_output

        cmd="kubectl exec -t " + name + " -n dualtor -- timeout 5s traceroute -n " + self.config.client_ip
        print cmd
        server_output=subprocess.check_output(cmd, shell=True, stderr=subprocess.STDOUT)
        print server_output


    def packets_received(self, name, server_log, count, previous_seq):
        error = 0

        if len(server_log) == 0: 
            _log.exception("empty server log of %s at %d seconds", name, count)
            raise Exception("error empty server log")

        last_log = server_log[-1]
        if last_log.find("--") == -1:
            _log.exception("failed to parse server log of %s at %d seconds", name, count)
            raise Exception("error parsing server log")
    
        seq_string = last_log.split("--")[0]
        seq = int(seq_string)
        diff = seq - previous_seq

        _log.info("%d second -- %s %s packets received", count, diff, name)
        #check if packets received is more than 50 except for first and last iterations.
        if previous_seq != 0 and seq != self.config.total_packets and diff < 50: 
            error =1
            _log.error("server log of %s at %d seconds -- received %d packets, link broken", name, count, diff)
            if self.rb_error == 5:
                print "may stop for debug"
                #time.sleep(2)
  
        return seq, error
    
    def clean_up(self):
        names = ["ra-server", "rb-server"]
        for name in names:
            cmd_prefix="kubectl exec -n dualtor -t " + name + " -- "
            output=subprocess.check_output(cmd_prefix + "ps -a", shell=True, stderr=subprocess.STDOUT)
            if output.find("nc -l") != -1:
                output=subprocess.check_output(cmd_prefix + "killall nc", shell=True, stderr=subprocess.STDOUT)

    def run_single_test(self, case_name, client_func, break_func, restore_func):
        test_name = "<" + self.config.spec_name + " -- " + case_name + ">"
        _log.info("\nstart running test %s ...", test_name)

        ra_server_log=[]
        rb_server_log=[]
        self.start_server("ra-server", ra_server_log)
        self.start_server("rb-server", rb_server_log)

        time.sleep(1)
        client_func()

        count=0
        ra_previous_seq=0
        rb_previous_seq=0
        while (ra_previous_seq != self.config.total_packets or rb_previous_seq != self.config.total_packets) and count < self.config.timeout:
          time.sleep(1)
          count +=1

          ra_previous_seq, ra_error = self.packets_received("ra-server", ra_server_log, count, ra_previous_seq)
          rb_previous_seq, rb_error = self.packets_received("rb-server", rb_server_log, count, rb_previous_seq)

          if count == 5:
              break_func()
    
          if count == 15:
              restore_func()

          self.ra_error += ra_error
          self.rb_error += rb_error
        
        # cleanup server
        self.clean_up()

        if self.ra_error > self.config.max_errors:
            _log.exception("test %s : client to ra-server failover failed. error count %d.", test_name, self.ra_error)
            raise Exception("failover test failed")

        if self.rb_error > self.config.max_errors:
            _log.exception("test %s : client to rb-server failover failed. error count %d.", test_name, self.rb_error)
            raise Exception("failover test failed")

        _log.info("test %s completed.", test_name)

    def client_func_pod_ip(self):   
        self.start_client("ra-server", self.config.ra_server_ip, self.config.target_port)
        self.start_client("rb-server", self.config.rb_server_ip, self.config.target_port)

    def client_func_service_ip(self):      
        self.start_client("service-ip-server-0", self.config.ra_service_ip, self.config.service_port)
        self.start_client("service-ip-server-1", self.config.rb_service_ip, self.config.service_port)

    def client_func_node_port(self):      
        self.start_client("node-port-0", self.config.node_port_ip, self.config.ra_node_port)
        self.start_client("node-port-1", self.config.node_port_ip, self.config.rb_node_port)

    def link_func_break_tor2tor(self):      
        # break tor2tor link via eth1 of tor router.
        # client to rb-server is currently via client_rb_plane.
        cmd="docker exec bird-a" + self.config.client_rb_plane + " ip link set dev eth1 down"
        output=subprocess.check_output(cmd, shell=True, stderr=subprocess.STDOUT)
        _log.info("link function: break tor2tor")

    def link_func_restore_tor2tor(self):
        cmd="docker exec bird-a" + self.config.client_rb_plane + " ip link set dev eth1 up"
        output=subprocess.check_output(cmd, shell=True, stderr=subprocess.STDOUT)
        _log.info("link function: restore tor2tor")

    def link_func_break_tor2node(self):      
        # break tor2node link via eth0 of tor router.
        # client to rb-server is currently via client_rb_plane.
        cmd="docker exec bird-b" + self.config.client_rb_plane + " ip link set dev eth0 down"
        output=subprocess.check_output(cmd, shell=True, stderr=subprocess.STDOUT)
        _log.info("link function: break tor2node")

    def link_func_restore_tor2node(self):
        cmd="docker exec bird-b" + self.config.client_rb_plane + " ip link set dev eth0 up"
        output=subprocess.check_output(cmd, shell=True, stderr=subprocess.STDOUT)
        _log.info("link function: restore tor2node")

    def link_func_drop_client_node(self):      
        # drop packets silently on client node on interface to rb-server.
        cmd="docker exec kind-worker tc qdisc add dev " +  self.config.client_rb_dev + " root netem loss 100%"
        output=subprocess.check_output(cmd, shell=True, stderr=subprocess.STDOUT)
        _log.info("link function: drop client node")

    def link_func_restore_client_node(self):
        cmd="docker exec kind-worker tc qdisc del dev " +  self.config.client_rb_dev + " root"
        output=subprocess.check_output(cmd, shell=True, stderr=subprocess.STDOUT)
        _log.info("link function: restore client node")

    def link_func_drop_server_node(self):      
        # drop packets silently on rb-server node on interface to client.
        cmd="docker exec kind-worker3 tc qdisc add dev " +  self.config.rb_server_dev + " root netem loss 100%"
        output=subprocess.check_output(cmd, shell=True, stderr=subprocess.STDOUT)
        _log.info("link function: drop server node")

    def link_func_restore_server_node(self):
        cmd="docker exec kind-worker3 tc qdisc del dev " +  self.config.rb_server_dev + " root"
        output=subprocess.check_output(cmd, shell=True, stderr=subprocess.STDOUT)
        _log.info("link function: restore server node")

    def do_nothing(self):
        pass

    def test_failover_tor2tor(self):
        self.run_single_test("failover_tor2tor", self.client_func, self.link_func_break_tor2tor, self.link_func_restore_tor2tor)

    def test_failover_tor2node(self):
        self.run_single_test("failover_tor2node", self.client_func, self.link_func_break_tor2node, self.link_func_restore_tor2node)

    def test_failover_drop_client(self):
        self.run_single_test("failover_drop_client", self.client_func, self.link_func_drop_client_node, self.link_func_restore_client_node)

    def test_failover_drop_server(self):
        self.run_single_test("failover_drop_server", self.client_func, self.link_func_drop_server_node, self.link_func_restore_server_node)

    def test_basic_connection(self):
        self.run_single_test("basic_connection", self.client_func, self.do_nothing, self.do_nothing)


# FailoverCluster holds methods to setup/cleanup testing enviroment. 
class FailoverCluster(object):
    def setup(self):
        kubectl("create ns dualtor")

        # Create client, ra-server, rb-server and service. 
        kubectl("run --generator=run-pod/v1 client -n dualtor" +
                " --image busybox --labels='pod-name=client' " +
                " --overrides='{ \"apiVersion\": \"v1\", \"spec\": { \"nodeSelector\": { \"kubernetes.io/hostname\": \"kind-worker\" } } }'" +
                " --command /bin/sleep -- 3600")
        kubectl("run --generator=run-pod/v1 client-host -n dualtor" +
                " --image busybox --labels='pod-name=client-host' " +
                " --overrides='{ \"apiVersion\": \"v1\", \"spec\": { \"hostNetwork\": true, \"nodeSelector\": { \"kubernetes.io/hostname\": \"kind-worker\" } } }'" +
                " --command /bin/sleep -- 3600")
        kubectl("run --generator=run-pod/v1 ra-server -n dualtor" +
                " --image busybox --labels='pod-name=ra-server,app=server' " +
                " --overrides='{ \"apiVersion\": \"v1\", \"spec\": { \"nodeSelector\": { \"kubernetes.io/hostname\": \"kind-control-plane\" } } }'" +
                " --command /bin/sleep -- 3600")
        kubectl("run --generator=run-pod/v1 rb-server -n dualtor" +
                " --image busybox --labels='pod-name=rb-server,app=server' " +
                " --overrides='{ \"apiVersion\": \"v1\", \"spec\": { \"nodeSelector\": { \"kubernetes.io/hostname\": \"kind-worker3\" } } }'" +
                " --command /bin/sleep -- 3600")
        kubectl("wait --timeout=1m --for=condition=ready" +
                " pod/client -n dualtor")
        kubectl("wait --timeout=1m --for=condition=ready" +
                " pod/client-host -n dualtor")
        kubectl("wait --timeout=1m --for=condition=ready" +
                " pod/ra-server -n dualtor")
        kubectl("wait --timeout=1m --for=condition=ready" +
                " pod/rb-server -n dualtor")


        # Create service
        self.create_service("ra-server")
        self.create_service("rb-server")

    def cleanup(self):
        kubectl("delete ns dualtor")

    def create_service(self, name):
        service = client.V1Service(
            metadata=client.V1ObjectMeta(
                name=name,
                labels={"name": name},
            ),
            spec={
                "ports": [{"port": 8090}],
                "selector": {"pod-name": name},
                "type": "NodePort",
            }
        )
        config.load_kube_config(os.environ.get('KUBECONFIG'))
        api_response = client.CoreV1Api().create_namespaced_service(
            body=service,
            namespace="dualtor",
        )

class TestFailoverPodIP(TestBase):
    @classmethod
    def setUpClass(cls):
        cluster = FailoverCluster()
        cluster.setup()

    @classmethod
    def tearDownClass(cls):
        cluster = FailoverCluster()
        cluster.cleanup()

    def setUp(self):
        TestBase.setUp(self)

        config = FailoverTestConfig(2000, "pod ip", 3)
        self.test = FailoverTest(config)

    def tearDown(self):
        pass
    def test_basic_connection(self):
        self.test.test_basic_connection()

    def test_failover_tor2tor(self):
        self.test.test_failover_tor2tor()

    def test_failover_tor2node(self):
        self.test.test_failover_tor2node()

    def test_failover_drop_client(self):
        self.test.test_failover_drop_client()

    def test_failover_drop_server(self):
        self.test.test_failover_drop_server() 

class TestFailoverServiceIP(TestBase):
    @classmethod
    def setUpClass(cls):
        cluster = FailoverCluster()
        cluster.setup()

    @classmethod
    def tearDownClass(cls):
        cluster = FailoverCluster()
        cluster.cleanup()

    def setUp(self):
        TestBase.setUp(self)

        config = FailoverTestConfig(2000, "service ip", 3)
        self.test = FailoverTest(config)

    def tearDown(self):
        pass

    def test_basic_connection(self):
        self.test.test_basic_connection()

    def test_failover_tor2tor(self):
        self.test.test_failover_tor2tor()

    def test_failover_tor2node(self):
        self.test.test_failover_tor2node()

    def test_failover_drop_client(self):
        self.test.test_failover_drop_client()

    def test_failover_drop_server(self):
        self.test.test_failover_drop_server() 

    def test_z_deploy_daemonset(self):
        # make it last test case to run. 
        # deploy four pods onto four nodes
        nodes = ["kind-control-plane", "kind-worker", "kind-worker2", "kind-worker3"]
        for node in nodes:
            kubectl("run --generator=run-pod/v1 " + node + " -n dualtor --image busybox " +
                    " --overrides='{ \"apiVersion\": \"v1\", \"spec\": { \"nodeSelector\": { \"kubernetes.io/hostname\": \"" + node + "\" } } }'" +
                    " --command /bin/sleep -- 3600")
            kubectl("wait --timeout=1m --for=condition=ready" +
                " pod/" + node + " -n dualtor")

class TestFailoverNodePort(TestBase):
    @classmethod
    def setUpClass(cls):
        cluster = FailoverCluster()
        cluster.setup()

    @classmethod
    def tearDownClass(cls):
        cluster = FailoverCluster()
        cluster.cleanup()

    def setUp(self):
        TestBase.setUp(self)
        # Node Port takes longer time to failover.
        config = FailoverTestConfig(2000, "node port", 3)
        self.test = FailoverTest(config)

    def tearDown(self):
        pass

    def test_basic_connection(self):
        self.test.test_basic_connection()

    def test_failover_tor2tor(self):
        self.test.test_failover_tor2tor()

    def test_failover_tor2node(self):
        self.test.test_failover_tor2node()

    def test_failover_drop_client(self):
        self.test.test_failover_drop_client()

    def test_failover_drop_server(self):
        self.test.test_failover_drop_server() 

class TestFailoverHostAccess(TestBase):
    @classmethod
    def setUpClass(cls):
        cluster = FailoverCluster()
        cluster.setup()

    @classmethod
    def tearDownClass(cls):
        cluster = FailoverCluster()
        cluster.cleanup()

    def setUp(self):
        TestBase.setUp(self)

        # host access takes longer time to failover.
        config = FailoverTestConfig(2000, "host access", 3)
        self.test = FailoverTest(config)

    def tearDown(self):
        pass

    def test_basic_connection(self):
        self.test.test_basic_connection()

    def test_failover_tor2tor(self):
        self.test.test_failover_tor2tor()

    def test_failover_tor2node(self):
        self.test.test_failover_tor2node()

    def test_failover_drop_client(self):
        self.test.test_failover_drop_client()

    def test_failover_drop_server(self):
        self.test.test_failover_drop_server() 
