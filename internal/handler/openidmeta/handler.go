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
		handler.AllowMethods(handler.StoreRequest(log, h.store,
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
		handler.AllowMethods(handler.StoreRequest(log, h.store,
			func(data openidmeta.Data) []byte { return data.JWKS },
		),
			log, http.MethodGet, http.MethodHead,
		),
	)
}
