// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-jose/go-jose/v4"
)

// ErrProjShootUIDInvalidFormat is an error that is returned if
// an issuer metadata shoot secret name is not in the correct format.
var ErrProjShootUIDInvalidFormat = errors.New("input not in the correct format: projectName--shootUID")

// SplitProjectNameAndShootUID splits the key by '--' in two parts.
func SplitProjectNameAndShootUID(key string) (string, string, error) {
	split := strings.Split(key, "--")
	if len(split) != 2 || strings.TrimSpace(split[0]) == "" || strings.TrimSpace(split[1]) == "" {
		return "", "", ErrProjShootUIDInvalidFormat
	}
	return split[0], split[1], nil
}

// LoadKeySet parses the jwks key set.
func LoadKeySet(jwks []byte) (*jose.JSONWebKeySet, error) {
	keySet := &jose.JSONWebKeySet{}
	if err := json.Unmarshal(jwks, &keySet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWKS: %w", err)
	}

	return keySet, nil
}

// OpenIDMetadata is a minimal struct allowing to parse the issuer and jwks URIs
// from the OIDC discovery page.
type OpenIDMetadata struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

// LoadOpenIDConfig parses the openid configuration page.
func LoadOpenIDConfig(config []byte) (*OpenIDMetadata, error) {
	openIDConfig := &OpenIDMetadata{}
	if err := json.Unmarshal(config, openIDConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal openid configuration: %w", err)
	}
	return openIDConfig, nil
}
