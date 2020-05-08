#!/bin/bash
set -e

# this script is used by Netlify to construct docs.projectcalico.org
# to do this, it builds several different sites:
# - latest
# - master
# - legacy-release
# - each archive release

if [ -z "$CURRENT_RELEASE" ]; then
    echo "must set \$CURRENT_RELEASE"
    exit 1
fi

if [ -z "$(which jekyll)" ]; then
    gem install github-pages
fi

if [ -z "$(which helm)" ]; then
    make bin/helm
    export PATH=$PATH:$(pwd)/bin
fi

DESTINATION=$(pwd)/_site

# If this is a deploy-preview, correctly set the URL to the generated Netlify subdomain
JEKYLL_CONFIG=_config.yml
if [ "$CONTEXT" == "deploy-preview" ]; then
    echo "url: $DEPLOY_PRIME_URL" >_config_url.yml
    JEKYLL_CONFIG=$JEKYLL_CONFIG,$(pwd)/_config_url.yml
fi

# As a side effect of choosing the a different release branching scheme
# only in private docs, we need to pick between the "release" prefix and
# "release-calient" prefix for release branch schemes. This function is
# for use when building non-legacy branches only.
# v2.7, v2.8 will "return" release- and everything else release-calient.
# Returns the prefix without a trailing "-".
function get_release_prefix() {
    local release_stream=$1
    if [[ ${release_stream} == v2.* ]]; then
        echo "release"
    else
        echo "release-calient"
    fi
}

# build builds a branch $1 into the dir _site/$2. If $2 is not provided, the
# branch is built into _site/
function build() {
    echo "[DEBUG] building branch $1 into dir $2"
    TEMP_DIR=$(mktemp -d)

    git clone --depth=1 https://$GH_DEPLOY_TOKEN@github.com/tigera/calico-private -b $1 $TEMP_DIR

    pushd $TEMP_DIR
    jekyll build --config $JEKYLL_CONFIG,$EXTRA_CONFIG --baseurl=$2 --destination _site/$2
    popd

    rsync -r $TEMP_DIR/_site .
}

# build_master builds skip the git clone and build the site in the current tree
function build_master() {
    jekyll build --config $JEKYLL_CONFIG --baseurl /master --destination _site/master
}

# build_archives builds the archives. The release-legacy branch is special
# and is built into _site directly (the legacy docs were a version per dir).
# Newer archive versions are built into its own directory for that version.
function build_archives() {
    grep -oP '^- \K(.*)' _data/archives.yml | while read branch; do
        EXTRA_CONFIG=$EXTRA_CONFIG,$(pwd)/netlify/_config_noindex.yml
        if [[ "$branch" == legacy* ]]; then
            if [ -z "$CUSTOM_ARCHIVE_PATH" ]; then
                build release-legacy
            else
                build release-legacy $CUSTOM_ARCHIVE_PATH
                EXTRA_CONFIG=$EXTRA_CONFIG,$(pwd)/netlify/_manifests_only.yml build release-legacy /
            fi
        else
            release_prefix=$(get_release_prefix ${branch})
            if [ -z "$CUSTOM_ARCHIVE_PATH" ]; then
                build ${release_prefix}-${branch} /${branch}
            else
                build ${release_prefix}-${branch} $CUSTOM_ARCHIVE_PATH/${branch}
                EXTRA_CONFIG=$EXTRA_CONFIG,$(pwd)/netlify/_manifests_only.yml build ${release_prefix}-${branch} /${branch}
            fi
        fi
    done
}

echo "[INFO] building master site"
build_master > /tmp/master.log 2>&1 &

echo "[INFO] building archives"
build_archives > /tmp/archives.log 2>&1 &

echo [INFO] Waiting for master and archive builds to complete: `date` ...
wait
echo [INFO] Master and archive builds complete: `date`.

echo Output from master build ...
cat /tmp/master.log

echo Output from archive build ...
cat /tmp/archives.log

mv _site/sitemap.xml _site/release-legacy-sitemap.xml

RELEASE_PREFIX=$(get_release_prefix $CURRENT_RELEASE)

echo "[INFO] building current release"
EXTRA_CONFIG=$(pwd)/netlify/_config_latest.yml build $RELEASE_PREFIX-$CURRENT_RELEASE
mv _site/sitemap.xml _site/latest-sitemap.xml

echo "[INFO] building permalink for current release"
EXTRA_CONFIG=$(pwd)/netlify/_config_latest.yml build $RELEASE_PREFIX-$CURRENT_RELEASE /$CURRENT_RELEASE

if [ ! -z "$CANDIDATE_RELEASE" ]; then
    RELEASE_PREFIX=$(get_release_prefix $CANDIDATE_RELEASE)
    echo "[INFO] building candidate release"
    build $RELEASE_PREFIX-$CANDIDATE_RELEASE /$CANDIDATE_RELEASE
fi

mv netlify/sitemap-index.xml _site/sitemap.xml
mv netlify/_redirects _site/_redirects
