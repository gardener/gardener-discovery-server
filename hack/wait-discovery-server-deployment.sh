#!/usr/bin/env bash

# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o pipefail
set -o errexit

# Waits for the MutatingAdmissionPolicy to be applied/removed and then
# triggers a rollout of the discovery-server deployment.

MODE="${1:-}"
repo_root="$(readlink -f $(dirname ${0})/..)"

case "$MODE" in
  deploy)
    container_image=$(cat "${repo_root}/example/local/discovery-server/discovery-server-image" | tr -d '\n')

    kubectl wait --timeout=30s --for=create \
        mutatingadmissionpolicies.admissionregistration.k8s.io/patch-discovery-server-image \
        mutatingadmissionpolicybinding.admissionregistration.k8s.io/patch-discovery-server-image

    # Looks like kube API server need some time to activate the MutatingAdmissionPolicy
    sleep 1

    # Use the annotation key 'kubectl.kubernetes.io/restartedAt' as any other will be removed by GRM.
    kubectl -n garden patch deployment gardener-discovery-server --type merge \
        -p "{\"spec\": {\"template\": {\"metadata\": {\"annotations\": {\"kubectl.kubernetes.io/restartedAt\": \"${container_image}\"}}}}}"
    kubectl -n garden rollout status deployment gardener-discovery-server --timeout=60s
    ;;

  delete)
    kubectl wait --timeout=30s --for=delete \
        mutatingadmissionpolicies.admissionregistration.k8s.io/patch-discovery-server-image \
        mutatingadmissionpolicybinding.admissionregistration.k8s.io/patch-discovery-server-image

    # Looks like kube API server need some time to deactivate the MutatingAdmissionPolicy
    sleep 1

    kubectl -n garden patch deployment gardener-discovery-server --type merge \
        -p '{"spec": {"template": {"metadata": {"annotations": {"kubectl.kubernetes.io/restartedAt": null}}}}}'

    kubectl -n garden rollout status deployment gardener-discovery-server --timeout=60s
    ;;

  *)
    echo "Usage: $0 {deploy|delete}" >&2
    exit 1
    ;;
esac
