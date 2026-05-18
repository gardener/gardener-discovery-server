#!/usr/bin/env bash

# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o pipefail
set -o errexit

repo_root="$(readlink -f "$(dirname "${0}")"/..)"

# Dump container image to a file so that other utility scripts can use it.
echo "${SKAFFOLD_IMAGE}" > "${repo_root}/example/local/discovery-server/discovery-server-image"

cat <<EOF > "${repo_root}/example/local/discovery-server/patch-mutating-admission-policy.yaml"
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingAdmissionPolicy
metadata:
  name: patch-discovery-server-image
spec:
  mutations:
    - patchType: JSONPatch
      jsonPatch:
        expression: |-
          [
            JSONPatch{
              op: "replace",
              path: "/spec/containers/0/image",
              value: "$SKAFFOLD_IMAGE"
            },
            JSONPatch{
              op: "add",
              path: "/metadata/labels/" + jsonpatch.escapeKey("discovery.gardener.cloud/image-patched"),
              value: "true"
            }
          ]
EOF
