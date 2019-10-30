{% unless include.autoscale == "true" %}
1. Modify the replica count in the `Deployment` named `calico-typha`
   to the desired number of replicas.

   ```
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: calico-typha
     ...
   spec:
     ...
     replicas: <number of replicas>
   ```
   {: .no-select-button}

   We recommend at least one replica for every 200 nodes and no more than
   20 replicas. In production, we recommend a minimum of three replicas to reduce
   the impact of rolling upgrades and failures.  The number of replicas should
   always be less than the number of nodes, otherwise rolling upgrades will stall.
   In addition, Typha only helps with scale if there are fewer Typha instances than
   there are nodes.
{% endunless %}

1. Generate TLS certificates for Felix and Typha to use to communicate. The following example
   uses OpenSSL to generate a CA, keys, and certificates, but you may generate them using any 
   X.509-compatible tool or obtain them from your organization's Certificate Authority. 

   ```bash
   openssl req -x509 -newkey rsa:4096 \
                     -keyout typhaca.key \
                     -nodes \
                     -out typhaca.crt \
                     -subj "/CN=Calico Typha CA" \
                     -days 3650
   openssl req -newkey rsa:4096 \
               -keyout typha.key \
               -nodes \
               -out typha.csr \
               -subj "/CN=calico-typha"
   openssl x509 -req -in typha.csr \
                     -CA typhaca.crt \
                     -CAkey typhaca.key \
                     -CAcreateserial \
                     -out typha.crt \
                     -days 3650
   openssl req -newkey rsa:4096 \
               -keyout felix.key \
               -nodes \
               -out felix.csr \
               -subj "/CN=calico-felix"
   openssl x509 -req -in felix.csr \
                     -CA typhaca.crt \
                     -CAkey typhaca.key \
                     -out felix.crt \
                     -days 3650
   ```
   > **Note**: The above example certificates are valid for 10 years. You are encouraged to choose a shorter
   > window of validity (365 days is typical), but note that you *must* rotate the certificates before they expire
   > or {{site.prodname}} will no longer function.
   {: .alert .alert-info}

1. Copy the certificates into the `calico.yaml` file. You will find five places in `calico.yaml` with text in angle
   brackets like `<replace with ....>`. Replace each one with the corresponding strings. The Certificate Authority
   is copied in verbatim, since it is in a ConfigMap, but the certificates and keys need to be base64 encoded since
   they are in Secrets. The following example command does this replacement using `sed` and the base64 encoding using
   the `base64` utility available on many Linux distributions.

   ```bash
   sed -e "s/<replace with base64-encoded Typha certificate>/$(cat typha.crt | base64 -w 0)/" \
       -e "s/<replace with base64-encoded Typha private key>/$(cat typha.key | base64 -w 0)/" \
       -e "s/<replace with base64-encoded Felix certificate>/$(cat felix.crt | base64 -w 0)/" \
       -e "s/<replace with base64-encoded Felix private key>/$(cat felix.key | base64 -w 0)/" \
       -i calico.yaml 
   sed -e 's/^/    /' < typhaca.crt > typhaca_indented.crt
   sed '/<replace with PEM-encoded (not base64) Certificate Authority bundle>/ {
     r typhaca_indented.crt
     d
   }' -i calico.yaml
   rm typhaca_indented.crt  
   ```
