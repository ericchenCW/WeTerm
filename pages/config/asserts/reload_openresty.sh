#!/bin/bash
set -euxo pipefail
systemctl restart consul-template
/usr/local/openresty/nginx/sbin/nginx -s reload