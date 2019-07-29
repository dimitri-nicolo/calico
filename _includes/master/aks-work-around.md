
## Create cluster

> **Note**: AKS will support cluster creation with the Azure CNI plugin in transparent mode in a future release. This section demonstrates the manual steps required in order to work around this limitation. 
Replace command parameters for each azure CLI command below with your own set of values.
{: .alert .alert-info}

1. Create an AKS cluster with Azure CNI without network policy support.

   **Example**
   ```
   az aks create \
   --resource-group myResourceGroup \
   --name myCluster \
   --node-count 2 \
   --node-vm-size Standard_D2s_v3 \
   --enable-addons monitoring \
   --generate-ssh-keys \
   --service-principal ${USER} \
   --client-secret ${PASS} \
   --network-plugin azure
   ```
   
1. Get cluster credentials. Make sure cluster is up.

   **Example**
   ```
   az aks get-credentials --resource-group myResourceGroup --name myCluster
   kubectl get node
   ```   

1. Create `bridge2transparent` daemonset to set Azure CNI plugin operating in transparent mode. 


   **Example**
   ```
   kubectl apply -f https://raw.githubusercontent.com/jonielsen/istioworkshop/master/03-TigeraSecure-Install/bridge-to-transparent.yaml
   ```  

   - **Info**: Please note nodes will be rebooted one by one once the manifest is applied. The node 
   in the middle of the reboot process is in `SchedulingDisabled` status. Wait till all nodes are in `Ready`
   status before taking next step.
     
