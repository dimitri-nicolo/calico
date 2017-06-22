---
title: Policy Audit Mode
---

Tigera Networking Essentials adds a Felix option DropActionOverride that configures how the
`deny` `action` in a [Rule]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/policy#Rule) is interpreted.
It can add logs for denied packets, or even allow the traffic through.

See the
[Felix configuration reference]({{site.baseurl}}/{{page.version}}/reference/felix/configuration#essentials-specific-configuration) for
information on how to cnofigure this option.

DropActionOverride controls what happens to each packet that is denied by
the current Calico policy - i.e. by the ordered combination of all the
configured policies and profiles that apply to that packet.  It may be
set to one of the following values:

- DROP
- ACCEPT
- LOG-and-DROP
- LOG-and-ACCEPT.

Normally the "DROP" or "LOG-and-DROP" value should be used, as dropping a
packet is the obvious implication of that packet being denied.  However when
experimenting, or debugging a scenario that is not behaving as you expect, the
"ACCEPT" and "LOG-and-ACCEPT" values can be useful: then the packet will be
still be allowed through.

When one of the "LOG-and-..." values is set, each denied packet is logged in
syslog, with an entry like this:

```
May 18 18:42:44 ubuntu kernel: [ 1156.246182] calico-drop: IN=tunl0 OUT=cali76be879f658 MAC= SRC=192.168.128.30 DST=192.168.157.26 LEN=60 TOS=0x00 PREC=0x00 TTL=62 ID=56743 DF PROTO=TCP SPT=56248 DPT=80 WINDOW=29200 RES=0x00 SYN URGP=0 MARK=0xa000000
```

Note that [Denied Packet Metrics]({{site.baseurl}}/{{page.version}}/reference/essentials/policy-violations) are independent of the DropActionOverride
setting.  Specifically, if packets that would normally be denied are being
allowed through by a setting of "ACCEPT" or "LOG-and-ACCEPT", those packets
still contribute to the denied packet metrics as normal.

One way to configure _DropActionOverride_ , would be to use
[calicoctl config command](../calicoctl/commands/config). For example,
to set a DropActionOverride for "myhost" to log then drop denied packets, use the
following command:

```
    ./calicoctl config set --raw=felix --node=myhost DropActionOverride LOG-and-DROP
```
