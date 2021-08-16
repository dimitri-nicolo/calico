> Felix can be configured to only manage a certain range of routing table indexes, via the routeTableRange property.
> The Tigera operator will set a range of 65-99 when running on a cluster using AWS CNI, so that it doesn't interfere with routing table indexes used by AWS CNI.
> If you have manually set the routeTableRange property, the operator will respect that.
> 
> The details of routing table indexes used by AWS CNI are documented {% include open-new-window.html text='here' url='https://docs.aws.amazon.com/eks/latest/userguide/pod-networking.html' %}.
> AWS CNI provisions multiple ENIs per node as the number of pods on the node increases. AWS CNI will add entries for the primary ENI into the main routing table, and will then create routing tables for each additional ENI, starting at index 2. Additionally, if VLANs are being used, it appears that AWS CNI will use tables from 100 onwards.
> 
> To check what the current routeTableRange value is in your cluster you can do the following:
> ```bash
> kubectl get felixconfiguration default -o yaml
> ```
> If routeTableRange is not present in the config, the value used will be 1-250. If the value is not 65-99, you may experience connectivity issues between pods due to missing routing tables and should set the value manually using the steps below.
>
> **Note**: The following steps will result in loss of connectivity between some pods. It is recommended to only make such changes during a maintenance window.
> To manually set the routeTableRange, you must do the following:
> 
> 1. Configure Felix to manage a routing table range which is distinct from the range used by AWS CNI:
>     ```bash
>     kubectl patch felixconfiguration default --type='merge' -p '{"spec": {"routeTableRange":{"min": 65, "max": 99}}}'
>     ````
>
> 1. Delete any routing rules and tables in the range 1-64 as they could be damaged or incomplete.
> 
> 1. Kill all the aws-node pods, which will force AWS CNI to recreate its routing rules and tables.
{: .alert .alert-info}
