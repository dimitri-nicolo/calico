---
title: calicoctl version
<<<<<<< HEAD
canonical_url: https://docs.tigera.io/v2.3/reference/calicoctl/commands/version
=======
redirect_from: latest/reference/calicoctl/commands/version
canonical_url: 'https://docs.projectcalico.org/v3.5/reference/calicoctl/commands/version'
>>>>>>> open/master
---

This section describes the `calicoctl version` command.

Read the [calicoctl Overview]({{site.baseurl}}/{{page.version}}/reference/calicoctl/)
for a full list of calicoctl commands.

## Displaying the help text for 'calicoctl version' commands

Run `calicoctl version --help` to display the following help menu for the
commands.

```
Usage:
  calicoctl version [--config=<CONFIG>]

Options:
  -h --help             Show this screen.
  -c --config=<CONFIG>  Path to the file containing connection configuration in
                        YAML or JSON format.
                        [default: /etc/calico/calicoctl.cfg]

Description:
  Display the version of calicoctl.
```
{: .no-select-button}

### Example

Use `calicoctl version` to obtain the following data.

{% include {{page.version}}/calicoctl-version.md %}

\* To obtain these values, you must configure `calicoctl`
<<<<<<< HEAD
   [to connect to your datastore](/{{page.version}}/usage/calicoctl/configure/).
=======
   [to connect to your datastore](/{{page.version}}/getting-started/calicoctl/configure/).
>>>>>>> open/master


## See also

<<<<<<< HEAD
-  [calicoctl configuration]({{site.baseurl}}/{{page.version}}/usage/calicoctl/configure/)
   for details on configuring `calicoctl` to access the {{site.prodname}} datastore.
=======
-  [Installing calicoctl]({{site.baseurl}}/{{page.version}}/getting-started/calicoctl/install).
>>>>>>> open/master
