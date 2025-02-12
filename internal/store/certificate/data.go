// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package certificate

import "github.com/gardener/gardener-discovery-server/internal/store"

var (
	_ store.Reader[Data] = (*store.Store[Data])(nil)
	_ store.Writer[Data] = (*store.Store[Data])(nil)
)

// Data holds public certificates.
type Data struct {
	CABundle []byte
}

// Copy returns a deep copy of [Data].
func Copy(data Data) Data {
	out := Data{
		CABundle: make([]byte, len(data.CABundle)),
	}
	copy(out.CABundle, data.CABundle)
	return out
}
