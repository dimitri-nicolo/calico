#!/bin/bash

set -e
#set -x

project="$1"
filter="$2"
num_jobs=25

if [ "${project}" = "" ]; then
  echo "Please specify a project as the first argument."
  exit 1
fi

if [ "${filter}" = "" ]; then
  echo "Please specify a node filter (e.g. 'gke-foo-') as the second argument."
  exit 1
fi

# Waits until there are fewer than $1 jobs running.
wait_for_jobs() {
  while test $(jobs -p | wc -w) -ge "$1"; do wait -n; done
}

gather_logs() {
  instance=$1
  gcloud compute ssh --project ${project} ${instance} -- -T <<EOF 2>&1 | gzip > "logs/${instance}.tmp"
echo docker ps
echo
docker ps
docker ps | grep -v pause | cut -d ' ' -f 1 | xargs -n1 -I_ sh -c "echo; echo Container _:; echo; docker logs _ 2>&1"
echo
echo Journal:
echo
sudo journalctl
EOF
  if [ $? = "0" ]; then
    mv "logs/${instance}.tmp" "logs/${instance}.log.gz"
    printf '.'
  else
    printf 'f'
  fi
}

nodes="$(gcloud compute instances list --project=${project} | cut -d ' ' -f1 | grep ${filter})"

mkdir -p logs
echo
for node in ${nodes}; do
  if [ -e logs/${node}.log.gz ]; then
    printf 'o'
    continue
  fi
  wait_for_jobs $num_jobs
  gather_logs ${node} &
  sleep 0.2
done
wait
echo
