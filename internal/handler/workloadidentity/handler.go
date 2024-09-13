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

	"github.com/gardener/gardener-discovery-server/internal/handler"
	"github.com/gardener/gardener-discovery-server/internal/utils"
)

const (
	headerCacheControl = "Cache-Control"
	pubCacheControl    = "public, max-age=3600"

	headerContentType = "Content-Type"
	mimeAppJSON       = "application/json"
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

// HandleOpenIDConfiguration handles /.well-known/openid-configuration.
func (h *Handler) HandleOpenIDConfiguration() http.Handler {
	log := h.log.WithName("openid-configuration")
	return handler.SetHSTS(
		handler.AllowMethods(handleRequest(log, h.oidc),
			log, http.MethodGet, http.MethodHead,
		),
	)
}

// HandleJWKS handles JWKS response.
func (h *Handler) HandleJWKS() http.Handler {
	log := h.log.WithName("jwks")
	return handler.SetHSTS(
		handler.AllowMethods(handleRequest(log, h.jwks),
			log, http.MethodGet, http.MethodHead,
		),
	)
}

func handleRequest(log logr.Logger, responseData []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerCacheControl, pubCacheControl)
		w.Header().Set(headerContentType, mimeAppJSON)

		if _, err := w.Write(responseData); err != nil {
			log.Error(err, "Failed writing response")
			return
		}
	})
}
