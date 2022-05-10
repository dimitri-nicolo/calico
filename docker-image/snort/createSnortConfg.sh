#!/bin/bash

SNORT_CONFIG=$1

cat >>${SNORT_CONFIG} <<EOF
# path to dynamic preprocessor libraries
dynamicpreprocessor directory /usr/lib64/snort-${SNORT_VERSION}_dynamicpreprocessor/

# path to base preprocessor engine
dynamicengine /usr/lib64/snort-${SNORT_VERSION}_dynamicengine/libsf_engine.so
EOF

rm "$0"

