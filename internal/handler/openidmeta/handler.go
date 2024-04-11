// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openidmeta

import (
	"net/http"

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	store "github.com/gardener/gardener-discovery-server/internal/store/openidmeta"
)

const (
	headerCacheControl = "Cache-Control"
	pubCacheControl    = "public, max-age=3600"

	headerContentType = "Content-Type"
	mimeAppJSON       = "application/json"

	responseNotFound   = `{"code":404,"message":"not found"}`
	responseBadRequest = `{"code":400,"message":"bad request"}`
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

// HandleWellKnown handles /.well-known/openid-configuration.
// It requires "projectName" and "shootUID" as path parameters.
func (h *Handler) HandleWellKnown(w http.ResponseWriter, r *http.Request) {
	shootUID := r.PathValue("shootUID")
	if _, err := uuid.Parse(shootUID); err != nil {
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(responseBadRequest)); err != nil {
			h.log.Error(err, "Failed writing response")
			return
		}
		return
	}

	projectName := r.PathValue("projectName")
	data, ok := h.store.Read(projectName + "--" + shootUID)
	if !ok {
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte(responseNotFound)); err != nil {
			h.log.Error(err, "Failed writing response")
			return
		}
		return
	}

	w.Header().Set(headerCacheControl, pubCacheControl)
	w.Header().Set(headerContentType, mimeAppJSON)
	if _, err := w.Write(data.Config); err != nil {
		h.log.Error(err, "Failed writing response")
		return
	}
}

// HandleJWKS handles JWKS response.
// It requires "projectName" and "shootUID" as path parameters.
func (h *Handler) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	shootUID := r.PathValue("shootUID")
	if _, err := uuid.Parse(shootUID); err != nil {
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(responseBadRequest)); err != nil {
			h.log.Error(err, "Failed writing response")
			return
		}
		return
	}

	projectName := r.PathValue("projectName")
	data, ok := h.store.Read(projectName + "--" + shootUID)
	if !ok {
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte(responseNotFound)); err != nil {
			h.log.Error(err, "Failed writing response")
			return
		}
		return
	}

	w.Header().Set(headerCacheControl, pubCacheControl)
	w.Header().Set(headerContentType, mimeAppJSON)
	if _, err := w.Write(data.JWKS); err != nil {
		h.log.Error(err, "Failed writing response")
		return
	}
}
