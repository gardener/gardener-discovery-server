# Gardener Discovery Server

[![reuse compliant](https://reuse.software/badge/reuse-compliant.svg)](https://reuse.software/)

A server capable of serving public metadata about different Gardener resources like shoot OIDC discovery documents.

## Development

As a prerequisite you need to have a Garden cluster up and running. The easiest way to get started is to follow the [Getting Started Locally Guide](https://github.com/gardener/gardener/blob/master/docs/development/getting_started_locally.md) which explains how to setup Gardener for local development.

Once the Garden cluster is up and running, export the `kubeconfig` pointing to the cluster as an environment variable.

```bash
export KUBECONFIG=/path-to/garden-kubeconfig
```

You should be able to start the discovery server with the following command.

```bash
make start
```
