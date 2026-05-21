#!/usr/bin/env bash

# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o pipefail
set -o errexit

# Waits for the MutatingAdmissionPolicy to be applied/removed and then
# triggers a rollout of the discovery-server deployment.

MODE="${1:-}"
repo_root="$(readlink -f "$(dirname "${0}")"/..)"

# Patches the deployment, waits for rollout, and ensures all pods have
# the correct patched/unpatched state by deleting mismatched pods.
# Arguments:
#   $1 - container image name (quoted) or null
#   $2 - mode, can be "deploy" or "delete"
patch_and_verify() {
  local annotation_value="$1"
  local mode="$2"

  local max_retries=5
  local retry_delay=2
  local pod_selector="app=gardener,role=discovery-server"
  local outdated_label_selector="discovery.gardener.cloud/image-patched"

  if [[ "$mode" == "deploy" ]]; then
    outdated_label_selector+="!=true"
  elif [[ "$mode" == "delete" ]]; then
    outdated_label_selector+="=true"
  else
    echo "ERROR: Invalid mode $mode"
    exit 1
  fi

  # re-use the annotation key 'kubectl.kubernetes.io/restartedAt' as any other will be removed by GRM.
  printf -v patch '{"spec": {"template": {"metadata": {"annotations": {"kubectl.kubernetes.io/restartedAt": %s}}}}}' "$annotation_value"

  kubectl -n garden patch deployment gardener-discovery-server --type merge -p "$patch"
  kubectl -n garden rollout status deployment gardener-discovery-server --timeout=60s

  for i in $(seq 1 $max_retries); do
    echo "Attempt $i/$max_retries: deleting mismatched pods"
    kubectl -n garden delete pods -l "${pod_selector},${outdated_label_selector}" --wait=true

    local mismatched_pods=$(kubectl -n garden get pods -l "${pod_selector},${outdated_label_selector}" -o jsonpath='{.items[*].metadata.name}')
    if [[ -z "$mismatched_pods" ]]; then
      echo "All pods are in the expected state."
      return 0
    fi

    if [[ $i -eq $max_retries ]]; then
      echo "ERROR: After $max_retries attempts, pods are not in the expected state." >&2
      echo "Mismatched pods: $mismatched_pods" >&2
      exit 1
    fi

    kubectl -n garden rollout status deployment gardener-discovery-server --timeout=60s
    sleep $retry_delay
  done
}

case "$MODE" in
  deploy)
    kubectl wait --timeout=30s --for=create \
        mutatingadmissionpolicies.admissionregistration.k8s.io/patch-discovery-server-image \
        mutatingadmissionpolicybinding.admissionregistration.k8s.io/patch-discovery-server-image

    container_image="\"$(cat "${repo_root}/example/local/discovery-server/discovery-server-image" | tr -d '\n')\""
    patch_and_verify "$container_image" "deploy"
    ;;

  delete)
    kubectl wait --timeout=30s --for=delete \
        mutatingadmissionpolicies.admissionregistration.k8s.io/patch-discovery-server-image \
        mutatingadmissionpolicybinding.admissionregistration.k8s.io/patch-discovery-server-image

    patch_and_verify "null" "delete"
    ;;

  *)
    echo "Usage: $0 {deploy|delete}" >&2
    exit 1
    ;;
esac
