#!/bin/sh
# Enable exit on non 0
set -e
set -x

iptables -t nat -A PREROUTING -p tcp -j REDIRECT --to-port 16001
iptables -t nat -A OUTPUT -p tcp -m owner ! --uid-owner 1729 -j REDIRECT --to-port 16002
