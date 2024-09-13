// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openidmeta

import (
	"net/http"

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	"github.com/gardener/gardener-discovery-server/internal/handler"
	store "github.com/gardener/gardener-discovery-server/internal/store/openidmeta"
)

const (
	headerCacheControl = "Cache-Control"
	pubCacheControl    = "public, max-age=3600"

	headerContentType = "Content-Type"
	mimeAppJSON       = "application/json"
)

var (
	responseBadRequest = []byte(`{"code":400,"message":"bad request"}`)
)

// Handler is capable or serving openid discovery documents.
type Handler struct {
	store store.Reader
	log   logr.Logger
}

// New constructs a new [Handler].
func New(store store.Reader, log logr.Logger) *Handler {
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
		handler.AllowMethods(handleRequest(log, h.store,
			func(data store.Data) []byte { return data.Config },
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
		handler.AllowMethods(handleRequest(log, h.store,
			func(data store.Data) []byte { return data.JWKS },
		),
			log, http.MethodGet, http.MethodHead,
		),
	)
}

func handleRequest(log logr.Logger, s store.Reader, getContent func(store.Data) []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shootUID := r.PathValue("shootUID")
		if _, err := uuid.Parse(shootUID); err != nil {
			w.Header().Set(headerContentType, mimeAppJSON)
			w.WriteHeader(http.StatusBadRequest)
			if _, err := w.Write(responseBadRequest); err != nil {
				log.Error(err, "Failed writing bad request response")
				return
			}
			return
		}

		projectName := r.PathValue("projectName")
		data, ok := s.Read(projectName + "--" + shootUID)
		if !ok {
			handler.NotFound(log).ServeHTTP(w, r)
			return
		}

		w.Header().Set(headerCacheControl, pubCacheControl)
		w.Header().Set(headerContentType, mimeAppJSON)
		if _, err := w.Write(getContent(data)); err != nil {
			log.Error(err, "Failed writing response")
			return
		}
	})
}
