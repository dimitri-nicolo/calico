#!/bin/bash

# This script uploads a binary to a private Azure blob and outputs a signed
# URL for the blob.  The signed URL can be used without being signed into
# Azure.
#
# The script requires the az command line to be installed and logged in.

file_to_upload="$1"
blob_name="$2"
resource_group="$3"
storage_account_name="$4"
container_name="$5"

storage_location=westus2

{
    echo "Creating the group..."
    az group create --location "${storage_location}" --resource-group "${resource_group}"

    echo "Creating the storage account..."
    az storage account create --location "${storage_location}" --name "${storage_account_name}" --resource-group "${resource_group}" --sku Standard_LRS

    echo "Getting the connection string..."
    export AZURE_STORAGE_CONNECTION_STRING="$(az storage account show-connection-string --name "${storage_account_name}" --resource-group "${resource_group}")"

    echo "Creating the container..."
    az storage container create --name "${container_name}"

    echo "Uploading the file..."
    az storage blob upload --container-name "${container_name}" --file "${file_to_upload}" --name "${blob_name}"

    expiry=`date +"%Y-%m-%dT%H:%M:%SZ" -d '30 days'`
    sas=`az storage blob generate-sas --container-name "${container_name}" --name "${blob_name}" --permissions r --https-only --expiry="${expiry}"`

    echo "Getting a URL to the file..."
    url=`az storage blob url --container-name "${container_name}" --name "${blob_name}" --sas-token ${sas}`

    echo "Full URL including access token:"
} 1>&2

echo "${url}?${sas}" | sed "s/\"//g"
