// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openidmeta

import "github.com/gardener/gardener-discovery-server/internal/store"

var (
	_ store.Reader[Data] = (*store.Store[Data])(nil)
	_ store.Writer[Data] = (*store.Store[Data])(nil)
)

// Data holds openid discovery metadata.
type Data struct {
	Config []byte
	JWKS   []byte
}

// Copy returns a deep copy of [Data].
func Copy(data Data) Data {
	out := Data{
		Config: make([]byte, len(data.Config)),
		JWKS:   make([]byte, len(data.JWKS)),
	}
	copy(out.Config, data.Config)
	copy(out.JWKS, data.JWKS)
	return out
}
