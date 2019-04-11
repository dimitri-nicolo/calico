#!/bin/sh

if $RUN_SCENARIO = true; then
  scaleloader -playbook-dir /playbooks /playbooks/scenario.yaml
else
  while /bin/true; do
    sleep 100
  done
fi

