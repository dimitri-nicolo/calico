#!/bin/bash
# This file executes htmlproofer checks on the compiled html files in _site.

# Version of htmlproofer to use.
HP_IMAGE=klakegg/html-proofer
HP_VERSION=3.19.2

# Local directories to ignore when checking external links
HP_IGNORE_LOCAL_DIRS="/_includes/"

# URLs to ignore when checking external links.
HP_IGNORE_URLS="/docs.openshift.org/,/openid.net/specs/openid-connect-core-1_0.html,/datatracker.ietf.org/doc/html/rfc7938,/docs.openshift.com/,#,/github.com\/projectcalico\/calico\/releases\/download/,/v2.3/,/v2.4/,/v2.5/,/v2.6/,/eksctl.io/,/stackpoint.io/,/api.my-ocp-domain.com/,/docs.vmware.com/,/github.com\/projectcalico\/felix\/issues\/1347/"

# jekyll uid
JEKYLL_UID=${JEKYLL_UID:=`id -u`}

# The htmlproofer check is flaky, so we retry a number of times if we get a bad result.
# If it doesn't pass once in 10 tries, we count it as a failed check.
echo "Running a hard URL check against recent releases"
echo > allstderr.out
for i in `seq 1 10`; do
	echo "htmlproofer attempt #${i}"

	# This docker run command MUST NOT USE -ti, or the stderr redirect will fail and the script will ignore all errors.
	docker run -e JEKYLL_UID=${JEKYLL_UID} --rm -v $(pwd)/_site:/_site/ ${HP_IMAGE}:${HP_VERSION} /_site --file-ignore ${HP_IGNORE_LOCAL_DIRS} --assume-extension --check-html --empty-alt-ignore --url-ignore ${HP_IGNORE_URLS} --internal_domains "docs.tigera.io" 2>stderr.out

	# Store the RC for future use.
	rc=$?
	echo "htmlproofer rc: $rc"

	# Show the errors
	cat stderr.out

	# Extract all the unique failing links and append them to the errors file
	grep -v "/_site" stderr.out | grep "*  " | sort | uniq >> allstderr.out

	# If the command executed successfully, break out. Otherwise, retry.
	if [[ $rc == 0 ]]; then break; fi

	# Break if there are no links that have failed every time.
	! cat allstderr.out | sort | uniq -c | grep "$i "
	if [[ $? == 0 ]]; then break; fi

	# Otherwise, sleep a short period and then retry.
	echo "htmlproofer failed, retry in 10s"
	sleep 10
done

# Rerun htmlproofer across _all_ files, but ignore failure, allowing us to notice legacy docs issues without failing CI
echo "Running a soft check across all files"
docker run -e JEKYLL_UID=${JEKYLL_UID} --rm -v $(pwd)/_site:/_site/ ${HP_IMAGE}:${HP_VERSION} /_site --assume-extension --check-html --empty-alt-ignore --url-ignore "#"

# Find all links that failed all ten times
echo "Links that failed in each run"
! cat allstderr.out | sort | uniq -c | grep "10 "
let greprc=$?

# Clean up output files
rm stderr.out
rm allstderr.out

# Exit using the return code from the loop above.
exit $greprc
