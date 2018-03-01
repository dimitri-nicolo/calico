---
title: Getting started with CNX
---

## Obtain the private registry credentials

Contact your Tigera support representative to obtain a Docker configuration file 
in JSON format. When you open the file, it should look something like the following.

```json
{
  "auths": {
    "quay.io": {
      "auth": "<ROBOT-TOKEN-VALUE>",
      "email": ""
    }
  }
}
```

The file should be named `config.json`. It contains a robot account token that will allow you to retrieve the {{site.prodname}} images from the private Tigera repository.
   
## Choose your orchestrator

To get started using {{site.prodname}}, we recommend running
through one or more of the available tutorials linked below.

These tutorials will help you understand the different environment options when
using {{site.prodname}}.  In most cases we provide worked examples using manual setup on
your own servers, a quick set-up in a virtualized environment using Vagrant and
a number of cloud services.

- [{{site.prodname}} with Kubernetes](kubernetes)
- [{{site.prodname}} with OpenShift](openshift/installation)
- [Host protection](bare-metal/bare-metal)
