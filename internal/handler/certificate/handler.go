// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package certificate

import (
	"net/http"

	"github.com/go-logr/logr"

	"github.com/gardener/gardener-discovery-server/internal/handler"
	"github.com/gardener/gardener-discovery-server/internal/store"
	"github.com/gardener/gardener-discovery-server/internal/store/certificate"
)

// Handler is capable of serving shoot cluster CA bundles.
type Handler struct {
	store store.Reader[certificate.Data]
	log   logr.Logger
}

// New constructs a new [Handler].
func New(store store.Reader[certificate.Data], log logr.Logger) *Handler {
	return &Handler{
		store: store,
		log:   log,
	}
}

// HandleCABundle handles /cluster-ca.
// It requires "projectName" and "shootUID" as path parameters.
func (h *Handler) HandleCABundle() http.Handler {
	log := h.log.WithName("cluster-ca")
	return handler.SetHSTS(
		handler.AllowMethods(handler.StoreRequest(log, h.store,
			func(data certificate.Data) []byte { return data.CABundle },
		),
			log, http.MethodGet, http.MethodHead,
		),
	)
}
