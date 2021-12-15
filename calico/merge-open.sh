#!/bin/bash

# Warning: this script requires manual editing whenever there's a new Calico / CNX minor release.
# See below (2 places to modify).
# Also, the following may be useful for removing the hundreds of canonical_url conflicts:
# perl -0777 -i -pe 's/<<<<<<< HEAD\n\|{7} merged common ancestors\n(canonical_url: .*\n|redirect_from: .*\n){0,2}=======\n(canonical_url: .*\n|redirect_from: .*\n){0,2}>>>>>>> [a-z0-9]+\n//' `grep -ril "=======" *`
# It will strip all conflicts that are just us removing an updated title segment containing canonical_urls and / or redirect_froms.  (There are more title section conflicts, but these are a start.)
# Long term there should probably be some tooling to just sort those out for SEO...
set -e

PRIVATE_REMOTE=${PRIVATE_REMOTE:-origin}
OPEN_REMOTE=${OPEN_REMOTE:-open}
MERGE_COMMIT=${MERGE_COMMIT:-master}

if [ -n "$1" ]
then
    echo "Completing merge from ${OPEN_REMOTE} into ${PRIVATE_REMOTE}"
    read -p "Press enter to continue or Ctrl-C to abort:"

    # When a new minor release is cut, add a new section at the end of this block.
    echo "Moving directories back to their original locations"
    git mv v3.0 v2.0
    git mv _includes/v3.0 _includes/v2.0
    git mv _data/v3_0 _data/v2_0
    git mv v3.1 v2.1
    git mv _includes/v3.1 _includes/v2.1
    git mv _data/v3_1 _data/v2_1
    git mv v3.2 v2.2
    git mv _includes/v3.2 _includes/v2.2
    git mv _data/v3_2 _data/v2_2
    git mv v3.4 v2.3
    git mv _includes/v3.4 _includes/v2.3
    git mv _data/v3_4 _data/v2_3
    git mv v3.6 v2.4
    git mv _includes/v3.6 _includes/v2.4
    git mv _data/v3_6 _data/v2_4
    git commit -m "Merge finish: move CNX docs directories back"

    echo "Merge complete!  Please create and submit your branch!"
    exit 0
fi

echo "Merging ${OPEN_REMOTE} into ${PRIVATE_REMOTE}"

git status
echo "Please ensure git status is clean before continuing."
read -p "Press enter to continue or Ctrl-C to abort:"

echo "Checking remote URLs.  Use PRIVATE_REMOTE and OPEN_REMOTE env vars to override."
git fetch ${PRIVATE_REMOTE}
git fetch ${OPEN_REMOTE}
git remote get-url ${PRIVATE_REMOTE} | grep tigera/calico-private
git remote get-url ${OPEN_REMOTE} | grep projectcalico/calico
git checkout master
git reset --hard ${PRIVATE_REMOTE}/master
echo "Remote URLs checked."
read -p "Press enter to continue or Ctrl-C to abort:"

# When a new minor release is cut, add a new section at the top of this block.
echo "Moving CNX docs (up to v2.1) to respective Calico locations (up to v3.1)"
git mv v2.4 v3.6
git mv _includes/v2.4 _includes/v3.6
git mv _data/v2_4 _data/v3_6
git mv v2.3 v3.4
git mv _includes/v2.3 _includes/v3.4
git mv _data/v2_3 _data/v3_4
git mv v2.2 v3.2
git mv _includes/v2.2 _includes/v3.2
git mv _data/v2_2 _data/v3_2
git mv v2.1 v3.1
git mv _includes/v2.1 _includes/v3.1
git mv _data/v2_1 _data/v3_1
git mv v2.0 v3.0
git mv _includes/v2.0 _includes/v3.0
git mv _data/v2_0 _data/v3_0
git commit -m "Merge prep: move CNX docs directories to Calico locations"
echo "Merge prep: move CNX docs directories to Calico locations - done"
read -p "Press enter to continue or Ctrl-C to abort:"

echo "Merging open source docs"
if [ ${MERGE_COMMIT} == "master" ]
then
    git merge --no-ff --no-commit ${OPEN_REMOTE}/master || true
else
    git merge --no-ff --no-commit ${MERGE_COMMIT} || true
fi
echo "Merging open source docs - done"
read -p "Press enter to continue or Ctrl-C to abort:"

echo "Removing directories that were never in CNX"
for ii in v1.5 v1.6 v2.0 v2.1 v2.2 v2.3 v2.4 v2.5 v2.6 v3.3 v3.5
do
    git rm -rf ${ii} || true
    git rm -rf _includes/${ii} || true
    git rm -rf _data/$(echo ${ii} | sed 's/\./_/') || true
done

echo "Resolve and commit the merge conflicts and then run this script again passing the argument continue (i.e. ./merge-open.sh continue)"
exit 0
