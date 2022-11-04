#!/bin/bash
set -eux

elasticsearch_path="/usr/share/elasticsearch"
elasticsearch_config="$elasticsearch_path/config"
cacerts_bcfks="$elasticsearch_config/cacerts.bcfks"
elasticsearch_keystore="$elasticsearch_config/elasticsearch.keystore"


echo "Initializing JDK JKS cacerts to BCFKS..."

if [ ! -f "$cacerts_bcfks" ]; then
    "$elasticsearch_path/jdk/bin/keytool" \
        -destkeystore "$cacerts_bcfks" \
        -deststorepass ${KEYSTORE_PASSWORD} \
        -deststoretype bcfks \
        -importkeystore \
        -providerclass org.bouncycastle.jcajce.provider.BouncyCastleFipsProvider \
        -providerpath /usr/share/bc-fips/bc-fips.jar \
        -srckeystore "$elasticsearch_path/jdk/lib/security/cacerts" \
        -srcstorepass changeit \
        -srcstoretype jks

    echo "JDK JKS cacerts are converted successfully to BCFKS."
else
    echo "JDK BCFKS cacerts exist. Skipped."
fi


echo "Initializing Elasticsearch keystore..."

if [ ! -f "$elasticsearch_keystore" ]; then
    "$elasticsearch_path/bin/elasticsearch-keystore" create -p <<EOF
${KEYSTORE_PASSWORD}
${KEYSTORE_PASSWORD}
EOF

    echo "Elasticsearch keystore initialization successful."
else
    echo "Elasticsearch keystore exists. Skipped."
fi


echo "Chowning for user elasticsearch:elasticsearch "

chown elasticsearch:elasticsearch "$elasticsearch_keystore"

echo "Chowning successful."
