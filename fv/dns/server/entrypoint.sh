#!/usr/bin/env sh

echo ${PORT}

echo ${RECORDS}

./dns-server ${PORT} ${RECORDS}
