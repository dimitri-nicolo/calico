#!/bin/sh
awk '/guardian.crt/ {print $2}' "$1" | base64 -d - > /tmp/guardian.crt
awk '/guardian.key/ {print $2}' "$1" | base64 -d - > /tmp/guardian.key
awk '/voltron.crt/  {print $2}' "$1" | base64 -d - > /tmp/voltron.crt