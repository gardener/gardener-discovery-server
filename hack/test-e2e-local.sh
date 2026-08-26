#!/usr/bin/env bash

# SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

echo "> E2E Tests"

GO111MODULE=on ginkgo run --timeout=1h --v --show-node-events "$@"
