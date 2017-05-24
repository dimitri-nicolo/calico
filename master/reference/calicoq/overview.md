---
title: calicoq
---

The command line tool, `calicoq`, makes it easy to check your Calico security
policies.  It can be downloaded from the Tigera Essentials Google Drive folder.

## Configuration

calicoq works by querying the Calico datastore and needs configuration so that
it knows what type of datastore you are using - either etcdv2 or the Kubernetes
API - and where that is.  For this configuration calicoq uses exactly the same
setup as calicoctl, which means that:

- You can create a YAML or JSON config file, and specify that with calicoq's
  `-c` option.  This is the best option if you have already created that file
  for use with calicoctl.

- Or you can set environment variables to specify the datastore type and
  location: `DATASTORE_TYPE` and so on.

For more detail, see
[Configuring calicoctl]({{site.baseurl}}/{{page.version}}/reference/calicoctl/setup).

## Commands

The calicoq command line interface provides a number of policy inspection
commands to allow you to confirm that your security policies are configured
as intended.

- The [endpoint]({{site.baseurl}}/{{page.version}}/reference/calicoq/endpoint)
  command shows you the Calico profiles and policies that relate to a specified
  endpoint.
- The [eval]({{site.baseurl}}/{{page.version}}/reference/calicoq/eval) command
  displays the endpoints that a selector selects.
- The [host]({{site.baseurl}}/{{page.version}}/reference/calicoq/host) command
  displays the policies that apply to endpoints on a host.
- The [version]({{site.baseurl}}/{{page.version}}/reference/calicoq/version)
  command displays the version of the tool.

## General options

```
-c --config=<CONFIG>      Path to the file containing connection
                          configuration in YAML or JSON format.
                          [default: /etc/calico/calicoctl.cfg]
```
