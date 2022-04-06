#!/usr/bin/env bash

cp semaphore.yml.tpl semaphore.yml
cp semaphore.yml.tpl semaphore-scheduler.yml

sed -i "s/\${BRANCH_RESTRICTION}/branch =~ \'.*\'/g" semaphore.yml
sed -i "s/\${BRANCH_RESTRICTION}/true/g" semaphore-scheduler.yml