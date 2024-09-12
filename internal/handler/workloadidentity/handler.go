// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package workloadidentity

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-logr/logr"

	"github.com/gardener/gardener-discovery-server/internal/utils"
)

const (
	headerCacheControl = "Cache-Control"
	pubCacheControl    = "public, max-age=3600"

	headerContentType = "Content-Type"
	mimeAppJSON       = "application/json"

	responseMethodNotAllowed = `{"code":405,"message":"method not allowed"}`
)

// Handler implements handler functions for the openid configuration and JWKS endpoints.
type Handler struct {
	oidc []byte
	jwks []byte
	log  logr.Logger
}

// New creates new workload identity handler.
func New(openIDConfig, jwks []byte, logger logr.Logger) (*Handler, error) {
	conf, err := utils.LoadOpenIDConfig(openIDConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load openid configuration: %w", err)
	}

	issuerURL, err := url.Parse(conf.Issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse issuer url: %w", err)
	}
	if issuerURL.Scheme != "https" {
		return nil, errors.New("invalid issuer url scheme")
	}
	if issuerURL.RawQuery != "" {
		return nil, errors.New("issuer url must not contain query")
	}
	if issuerURL.Fragment != "" {
		return nil, errors.New("issuer url must not contain fragment")
	}

	jwksURL, err := url.Parse(conf.JWKSURI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jwks url: %w", err)
	}
	if jwksURL.Scheme != "https" {
		return nil, errors.New("invalid jwks url scheme")
	}

	keySet, err := utils.LoadKeySet(jwks)
	if err != nil {
		return nil, fmt.Errorf("failed to load json web key set: %w", err)
	}

	for _, k := range keySet.Keys {
		if !k.IsPublic() {
			return nil, fmt.Errorf("jwks key with id %q is not public", k.KeyID)
		}
	}

	return &Handler{
		oidc: openIDConfig,
		jwks: jwks,
		log:  logger,
	}, nil
}

// HandleWellKnown handles /.well-known/openid-configuration.
func (h *Handler) HandleWellKnown(w http.ResponseWriter, r *http.Request) {
	handleRequest(h.log.WithName("well-known"), h.oidc, w, r)
}

// HandleJWKS handles JWKS response.
func (h *Handler) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	handleRequest(h.log.WithName("jwks"), h.jwks, w, r)
}

func handleRequest(logger logr.Logger, responseData []byte, w http.ResponseWriter, r *http.Request) {
	if w.Header().Get("Strict-Transport-Security") == "" {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000")
	}

	if isValid := handleInvalidRequest(logger.WithName("invalid-request-handler"), w, r); !isValid {
		return
	}

	w.Header().Set(headerCacheControl, pubCacheControl)
	w.Header().Set(headerContentType, mimeAppJSON)

	if _, err := w.Write(responseData); err != nil {
		logger.Error(err, "Failed writing response")
		return
	}
}

// handleInvalidRequest handles invalid requests.
// It returns true if the request is valid, and false otherwise.
func handleInvalidRequest(logger logr.Logger, w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}

	w.Header().Set(headerContentType, mimeAppJSON)
	w.WriteHeader(http.StatusMethodNotAllowed)
	if _, err := w.Write([]byte(responseMethodNotAllowed)); err != nil {
		logger.Error(err, "Failed writing response")
		return false
	}
	return false
}
