
# Documentation for miscellaneous BPF programming details

## Egress gateway

`EGRESS_CLIENT` and `EGRESS_GATEWAY` flags are patched into TC program
for a pod iff:
- `EGRESS_CLIENT`: the pod is an egress client
- `EGRESS_GATEWAY`: the pod is an egress gateway.

(Note, it isn't allowed to be both, so as to avoid loops.)

Expected conntrack states:
- CT-A:
  - src = client pod IP
  - dst = external destination IP
- CT-B:
  - src = client node IP
  - dst = gateway pod IP
- CT-C:
  - src = gateway pod IP
  - dst = external destination IP

When client and gateway are on different nodes:

```
 +---------------------------------------------------------------------+
 | +---------+                                                         |
 | | Egress  | caliX   CT-A      egress.  CT-B   opt                   |
 | | client  |-------> Host --->  calico ----> cluster ----> eth0 -------->
 | | pod     | [1]                encap         encap                  |
 | |         |                                                         |
 | |         | caliX            CT-A       [4]   opt                   |
 | |         |<------- Host <----------------- cluster <---- eth0 <--------
 | |         |                                  decap                  |
 | +---------+                                                         |
 +---------------------------------------------------------------------+

   +-----------------------------------------------------------------+
   |                                   +---------+                   |
   |               opt      CT-B       |  Egress |     CT-C          |
 ----> eth0 ---> cluster --> Host -------> gw  -----> Host --> eth0 ---> destination
   |              decap                |   pod   |                   |
   |                                   | decap+  |                   |
   |                                   |  SNAT   |                   |
   |               opt      CT-A   [2] |         |     CT-C          |
 <---- eth0 <--- cluster <-- Host <------- DNAT<----- Host <-- eth0 <--- destination
   |              encap                +---------+                   |
   +-----------------------------------------------------------------+
```

Expected processing, different nodes:
- outbound:
  - client node client-E: miss, create CT-A
  - client node eth0/tunl-E: miss, create CT-B
  - gateway node eth0/tunl-I: miss, create CT-B
  - gateway node gateway-I: hit CT-B
  - gateway node gateway-E: miss, create CT-C
  - gateway node eth0-E: hit CT-C
- return:
  - gateway node eth0-I: hit CT-C
  - gateway node gateway-I: hit CT-C
  - gateway node gateway-E: miss, create CT-A
  - gateway node eth0/tunl-E: hit CT-A
  - client node eth0/tunl-I: hit CT-A
  - client node client-I: hit CT-A

(Wherever we say "miss, create", we must allow for hit as well, for
subsequent packets in same flow.)

[1] Outbound direction for EGRESS_CLIENT when destination is outside
the cluster:
- No FIB, i.e. drop to IP stack, and set EGRESS bit, so that `ip rule`
  will take effect to route packet via egress.calico device.
- Mark CT state as egress flow so that we do this for subsequent
  packets in a flow as well as for the first.

[2] On the return path, on egress from an egress gateway, when gateway
and client are on different nodes:
- CT miss is expected because the unencapped flow has not previously
  been seen here.
- This would normally be mid-flow TCP, handled by drop to iptables,
  but in the `EGRESS_GATEWAY` case we handle as `CT_NEW` and whitelist
  both sides, because otherwise a mid-flow TCP packet will be
  considered invalid on the `FROM_HOST` side (at the following host or
  tunnel interface leaving the gateway node).
- Skip tc.c RPF check, because we expect source IP to differ from the
  egress gateway pod's own IP, on the return path.
- Set `SKIP_RPF` mark when dropping to iptables, for same reason.

[4] On the return path of an egress gateway flow with client and
gateway on different nodes, TO_HOST from a cluster encap device:
- Set `SKIP-RPF` mark when dropping to iptables, because source IP is
  an external destination that would not normally be routed through
  the device that the packet just arrived through.

When client and gateway are on same node:

```
 +--------------------------------------------------------------------------+
 | +---------+                                +---------+                   |
 | | Egress  | caliX   CT-A      egress. CT-B |  Egress |     CT-C          |
 | | client  |-------> Host --->  calico -------> gw  -----> Host --> eth0 ---> destination
 | | pod     | [1]                encap       |   pod   |                   |
 | |         |                                | decap+  |                   |
 | |         |                                |  SNAT   |                   |
 | |         | caliX              CT-A    [3] |         |     CT-C          |
 | |         |<------- Host <-------------------- DNAT<----- Host <-- eth0 <--- destination
 | +---------+                                +---------+                   |
 +--------------------------------------------------------------------------+
```

Expected processing, same nodes:
- outbound:
  - client node client-E: miss, create CT-A
  - gateway node gateway-I: miss, create CT-B
  - gateway node gateway-E: miss, create CT-C
  - gateway node eth0-E: hit CT-C
- return:
  - gateway node eth0-I: hit CT-C
  - gateway node gateway-I: hit CT-C
  - gateway node gateway-E: hit CT-A
  - client node client-I: hit CT-A

[1] same as when client and gateway are on different nodes.

[3] On the return path, on egress from an egress gateway, when gateway
and client are on the same node:
- Skip tc.c RPF check, because we expect source IP to differ from the
  egress gateway pod's own IP, on the return path.
- Set `SKIP_RPF` mark when dropping to iptables, for same reason.
