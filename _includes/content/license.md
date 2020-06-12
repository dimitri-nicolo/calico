**Is there a difference between an evaluation and a commercial license?**
  
  No.

**How long does it take to get a new {{site.prodname}} license?**

  After submitting a sales purchase order to Tigera, about 1-2 days.

**What happens to nodes during the license grace period?**

  All {{site.prodname}} features work without interruption.

**How do I know the license grace period?**

  Grace period terms are communicated to you when you get your commercial license. 

**What happens to nodes after the license grace period?**

- Initially, users can access the {{site.prodname}} Manager, and a message is displayed to change the license. But after two days, users will not be able to access {{site.prodname}} Manager.
- Although components appear to function in {{site.prodname}} Manager when the grace period expires, functionality falls back to open source Calico behavior; no tiers, and policy enforcement is limited to the default Kubernetes tier.  

**What happens if I add nodes beyond what I'm licensed for?**

- Node limits are not currently enforced
- All {{site.prodname}} features still work

**How do I access information about my license?**

The best way is to [use Prometheus]({{site.baseurl}}/maintenance/monitor/license-agent) to monitor {{site.prodname}} license metrics and set alerts. (License metrics are not available in {{site.prodname}} Manager.)
