// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"crypto/tls"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("App", func() {
	Context("getCipherSuiteIDs", func() {
		It("should return the default cipher suite IDs excluding the ones with CBC mode", func() {
			cipherIDs := getCipherSuiteIDs()
			Expect(cipherIDs).ToNot(ContainElement(tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA))
			Expect(cipherIDs).ToNot(ContainElement(tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA))
			Expect(cipherIDs).ToNot(ContainElement(tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA))
			Expect(cipherIDs).ToNot(ContainElement(tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA))
			Expect(cipherIDs).ToNot(BeEmpty())
		})
	})
})
