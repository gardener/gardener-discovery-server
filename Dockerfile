# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

############# builder
FROM golang:1.23.4 AS builder

ARG TARGETARCH
WORKDIR /go/src/github.com/gardener/gardener-discovery-server

# Copy go mod and sum files
COPY go.mod go.sum ./
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

COPY . .

ARG EFFECTIVE_VERSION
RUN make install EFFECTIVE_VERSION=$EFFECTIVE_VERSION

############# gardener-discovery-server
FROM gcr.io/distroless/static-debian12:nonroot AS gardener-discovery-server
WORKDIR /

COPY --from=builder /go/bin/discovery-server /gardener-discovery-server
ENTRYPOINT ["/gardener-discovery-server"]
