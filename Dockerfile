# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

############# builder
FROM --platform=$BUILDPLATFORM golang:1.26.1 AS builder

WORKDIR /go/src/github.com/gardener/gardener-discovery-server

# Copy go mod and sum files
COPY go.mod go.sum ./
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

COPY . .

ARG EFFECTIVE_VERSION
ARG TARGETOS
ARG TARGETARCH
RUN make build EFFECTIVE_VERSION=$EFFECTIVE_VERSION GOOS=$TARGETOS GOARCH=$TARGETARCH BUILD_OUTPUT_FILE="/output/bin/"

############# gardener-discovery-server
FROM gcr.io/distroless/static-debian13:nonroot AS gardener-discovery-server
WORKDIR /

COPY --from=builder /output/bin/discovery-server /gardener-discovery-server
ENTRYPOINT ["/gardener-discovery-server"]
