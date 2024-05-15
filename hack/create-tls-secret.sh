#!/bin/bash

# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0
# 

set -o errexit
set -o pipefail

repo_root="$(readlink -f $(dirname ${0})/..)"

"$repo_root"/hack/cert-gen.sh

kubectl apply -f <(cat <<EOF
---
apiVersion: v1
kind: Secret
metadata:
  name: gardener-discovery-server-tls
  namespace: garden
type: Opaque
data:
  tls.key: $(base64 -w 0 "$repo_root"/example/local/certs/tls.key)
  tls.crt: $(base64 -w 0 "$repo_root"/example/local/certs/tls.crt)
EOF
)
