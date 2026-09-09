// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openidmeta

import (
	"net/http"

	"github.com/go-logr/logr"

	"github.com/gardener/gardener-discovery-server/internal/handler"
	"github.com/gardener/gardener-discovery-server/internal/store"
	"github.com/gardener/gardener-discovery-server/internal/store/openidmeta"
)

// pubCacheControl is the Cache-Control value served for service account
// discovery documents. It is intentionally short so that consumers refetch
// the JWKS promptly after a service account signing key rotation, since the
// Kubernetes OIDC authenticator does not refetch on an unknown key ID and only
// refreshes once the cached keys expire per this header.
// See https://github.com/kubernetes/kubernetes/issues/139769.
const pubCacheControl = "public, max-age=120"

// Handler is capable of serving openid discovery documents.
type Handler struct {
	store store.Reader[openidmeta.Data]
	log   logr.Logger
}

// New constructs a new [Handler].
func New(store store.Reader[openidmeta.Data], log logr.Logger) *Handler {
	return &Handler{
		store: store,
		log:   log,
	}
}

// HandleOpenIDConfiguration handles /.well-known/openid-configuration.
// It requires "projectName" and "shootUID" as path parameters.
func (h *Handler) HandleOpenIDConfiguration() http.Handler {
	log := h.log.WithName("openid-configuration")
	return handler.SetHSTS(
		handler.AllowMethods(handler.StoreRequest(log, h.store, pubCacheControl,
			func(data openidmeta.Data) []byte { return data.Config },
		),
			log, http.MethodGet, http.MethodHead,
		),
	)
}

// HandleJWKS handles JWKS response.
// It requires "projectName" and "shootUID" as path parameters.
func (h *Handler) HandleJWKS() http.Handler {
	log := h.log.WithName("jwks")
	return handler.SetHSTS(
		handler.AllowMethods(handler.StoreRequest(log, h.store, pubCacheControl,
			func(data openidmeta.Data) []byte { return data.JWKS },
		),
			log, http.MethodGet, http.MethodHead,
		),
	)
}
