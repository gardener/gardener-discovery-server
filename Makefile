# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

.PHONY: run
run:
	./hack/cert-gen.sh
	go run ./cmd/discovery-server/main.go --tls-cert-file=./example/local/certs/tls.crt --tls-private-key-file=./example/local/certs/tls.key

.PHONY: test
test:
	go test -race -cover ./...
