#!/bin/bash

write_disclaimer() {
  echo "# !! WARNING, DO NOT EDIT !! This file is generated from semaphore.yml.d." >"$1"
  echo "# To update, modify the template and then run 'make gen-semaphore-yaml'." >>"$1"
}

# generate semaphore yaml file for PR and nightly builds
for out_file in semaphore.yml semaphore-scheduled-builds.yml; do
  write_disclaimer $out_file

  cat semaphore.yml.d/01-preamble.yml >>$out_file
  cat semaphore.yml.d/02-global_job_config.yml >>$out_file
  cat semaphore.yml.d/03-promotions.yml >>$out_file

  echo "blocks:" >>$out_file
  cat semaphore.yml.d/blocks/*.yml >>$out_file
done

sed -i "s/\${FORCE_RUN}/false/g" semaphore.yml
sed -i "s/\${WEEKLY_RUN}/false/g" semaphore.yml
sed -i "s/\${FORCE_RUN}/true/g" semaphore-scheduled-builds.yml
sed -i "s/\${WEEKLY_RUN}/false/g" semaphore-scheduled-builds.yml

# generate semaphore yaml file for third-party builds
out_file=semaphore-third-party-builds.yml

write_disclaimer $out_file

cat semaphore.yml.d/01-preamble.yml >>$out_file
cat semaphore.yml.d/02-global_job_config.yml >>$out_file

echo "blocks:" >>$out_file
cat semaphore.yml.d/blocks/10-prerequisites.yml >>$out_file
cat semaphore.yml.d/blocks/30-deep-packet-inspection.yml >>$out_file
cat semaphore.yml.d/blocks/30-elasticsearch.yml >>$out_file
cat semaphore.yml.d/blocks/50-fluentd.yml >>$out_file

sed -i "s/\${FORCE_RUN}/true/g" semaphore-third-party-builds.yml
sed -i "s/\${WEEKLY_RUN}/true/g" semaphore-third-party-builds.yml
