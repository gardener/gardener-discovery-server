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

	responseBadRequest       = `{"code":400,"message":"bad request"}`
	responseNotFound         = `{"code":404,"message":"not found"}`
	responseMethodNotAllowed = `{"code":405,"message":"method not allowed"}`
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
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusMethodNotAllowed)
		if _, err := w.Write([]byte(responseMethodNotAllowed)); err != nil {
			h.log.Error(err, "Failed writing response")
			return
		}
		return
	}

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
		h.HandleNotFound(w, r)
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
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusMethodNotAllowed)
		if _, err := w.Write([]byte(responseMethodNotAllowed)); err != nil {
			h.log.Error(err, "Failed writing response")
			return
		}
		return
	}
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
		h.HandleNotFound(w, r)
		return
	}

	w.Header().Set(headerCacheControl, pubCacheControl)
	w.Header().Set(headerContentType, mimeAppJSON)
	if _, err := w.Write(data.JWKS); err != nil {
		h.log.Error(err, "Failed writing response")
		return
	}
}

// HandleNotFound writes a not found response to writer.
func (h *Handler) HandleNotFound(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(headerContentType, mimeAppJSON)
	w.WriteHeader(http.StatusNotFound)
	if _, err := w.Write([]byte(responseNotFound)); err != nil {
		h.log.Error(err, "Failed writing response")
		return
	}
}
