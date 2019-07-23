---
title: Previewing the Impact of Policy Changes
---

### Big Picture

Preview the impacts of policy changes before you apply them.

### Value

Create, update, and delete policy with confidence knowing you will not unintentionally expose or block other network traffic.

### Features

This guide uses the following {{ site.prodname }} features:

- The {{ site.prodname }} Manager

### Before you begin...

You must have a running kubernetes cluster with {{ site.prodname }} installed.

### How to

1. From the Edit Policy page on the Manager, modify any attribute of the policy. 
1. Before applying it, click the "preview" button at the top right. This will launch the flow log visualizer.
1. Click the "changes applied" toggle to see how flows would change if the changes were applied.
1. Click the "only changed flows" to hide all flows which remained the same before and after the change.
1. Click the left arrow at the top-right corner of the view to return to the edit policy page.

>**Note**: There may be some flows which {{ site.prodname }} will not be able to predict. Those flows will appear as "Uncertain" as per the legend at the bottom right.
{: .alert .alert-info}
