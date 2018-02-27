#!/bin/sh

# Script that inspects the jekyll-generated _site and emits
# a Google App Engine service specification file on stdout.

if [ ! -d _site ]; then
	echo _site/ doesn\'t exist yet, make sure you run \"jekyll build\".
	exit 1
fi

cat <<EOF
# A Google Application Engine "service" definition. The reference
# to python is a red-herring - we're just statically serving the
# contents of the _site subdirectory. GAE isn't setup to do this
# easily - we need to tell it how to handle each type of file, as
# well as infer "index.html" when we just refer to a site directory.
#
runtime: python27
api_version: 1
threadsafe: yes

EOF

# Get all the unique file suffixes in the _site; e.g. ".txt", ".png".
# Output in form "js|html|yml", etc.
suffixes=`find _site -type f -iname \*.* -print | sed 's/.*\.//' | sort | uniq | paste -sd "|" -`

# Create a static file handler based on all the suffixes
printf "# Serve up static files based on suffix, let GAE infer mime-type\n"
printf "#\n"
printf "handlers:\n"
printf -- "- url: /(.*\\.(%s))$\n" $suffixes
printf "  static_files: _site/\\\1\n"
printf "  upload: _site/(.*\\.(%s))$\n" $suffixes

# Get all the directories in the _site; e.g. "/v2.0/getting-started".
# Output in form "/dir1|/dir2", etc.
directories=`find _site -type d -print | sed 's/_site\///g' | sort | uniq | paste -sd "|" -`

# Create a handler for URLs with a directory that do NOT have a
# terminal /. This is a fail-safe in case someone misconstructs
# the URL without a terminal /.
printf "\n"
printf "# Handle all directories that are missing a terminal /\n"
printf "#\n"
printf -- "- url: /(%s)$\n" $directories
printf "  static_files: _site/\\\1/index.html\n"
printf "  upload: _site/\\\1/index.html\n"

cat <<EOF

# Default directory/html append rules
#
- url: /
  static_files: _site/index.html
  upload: _site/index.html

# For directories indicated by a terminal /, append "/index.html"
- url: /(.+)/
  static_files: _site/\1/index.html
  upload: _site/(.+)/index.html
  expiration: "15m"

# For md files, append ".html"
- url: /(.+[a-z0-9])
  static_files: _site/\1.html
  upload: _site/(.+[a-z]).html
  expiration: "15m"
- url: /(.+)
  static_files: _site/\1/index.html
  upload: _site/(.+)/index.html
  expiration: "15m"
- url: /(.*)
  static_files: _site/\1
  upload: _site/(.*)

libraries:
- name: webapp2
  version: "2.5.2"
EOF

