#!/bin/bash

set -e
[[ -n "$DEBUG" ]] && set -x

if [ "$(ls -A /usr/local/share/ca-certificates)" ]; then
  cat /usr/local/share/ca-certificates/* >> /etc/ssl/certs/ca-certificates.crt
fi

/bin/marathon-consul $@
