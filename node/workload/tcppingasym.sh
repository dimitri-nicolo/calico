#!/bin/sh
printf hello | nc -w1 $1 8080 | grep hellohello
