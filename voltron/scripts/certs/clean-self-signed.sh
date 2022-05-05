#!/bin/sh

OUT_DIR=$1

if [ -z $1 ] ; then
    OUT_DIR="."
fi
echo $OUT_DIR

BASE=`dirname $0`

rm -f $OUT_DIR/cert
rm -f $OUT_DIR/key*
