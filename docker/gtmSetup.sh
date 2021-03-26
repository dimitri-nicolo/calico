#!/bin/bash

if [ "$GTM_INTEGRATION" != 'enabled' ]
then
rm -rf /usr/share/kibana/src/plugins/google_tag_manager
fi

#remove this script from fs
rm "$0"
