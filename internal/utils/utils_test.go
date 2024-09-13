// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"errors"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/gardener-discovery-server/internal/utils"
)

var _ = Describe("Utils", func() {
	Describe("#SplitProjectNameAndShootUID", func() {
		It("should correctly split project name and shoot uid", func() {
			projName := "test"
			uid := uuid.New().String()
			name, id, err := utils.SplitProjectNameAndShootUID(projName + "--" + uid)
			Expect(err).ToNot(HaveOccurred())
			Expect(name).To(Equal(projName))
			Expect(id).To(Equal(uid))
		})

		DescribeTable(
			"should produce an error",
			func(input string) {
				name, id, err := utils.SplitProjectNameAndShootUID(input)
				Expect(errors.Is(err, utils.ErrProjShootUIDInvalidFormat)).To(BeTrue())
				Expect(name).To(BeEmpty())
				Expect(id).To(BeEmpty())
			},
			Entry("empty string", ""),
			Entry("just a delimiter", "--"),
			Entry("delimiter with prefix", "a--"),
			Entry("delimiter with suffix", "--a"),
			Entry("no delimiter", "foo"),
			Entry("too many arguments to split", "a--b--c"),
		)
	})

	Describe("#LoadOpenIDConfig", func() {
		It("should successfully load openid configuration", func() {
			rawOpenIDConfig := []byte(`{
    "issuer": "https://test.discovery-server.gardener.cloud/issuer",
    "jwks_uri": "https://test.discovery-server.gardener.cloud/issuer/jwks",
    "response_types_supported": [
        "id_token"
    ],
    "subject_types_supported": [
        "public"
    ],
    "id_token_signing_alg_values_supported": [
        "RS256"
    ]
}`)
			config, err := utils.LoadOpenIDConfig(rawOpenIDConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(config.Issuer).To(Equal("https://test.discovery-server.gardener.cloud/issuer"))
			Expect(config.JWKSURI).To(Equal("https://test.discovery-server.gardener.cloud/issuer/jwks"))
		})

		It("should fail to load invalid openid configuration", func() {
			invalidConfig := []byte(`invalid config`)
			_, err := utils.LoadOpenIDConfig(invalidConfig)
			Expect(err).To(MatchError(ContainSubstring("failed to unmarshal openid configuration")))
		})
	})

	Describe("#LoadKeySet", func() {
		It("should successfully load JSON Web Key Set", func() {
			rawJWKS := []byte(`{
    "keys": [
        {
            "use": "sig",
            "kty": "RSA",
            "kid": "2qAiW7sYZexfKki3E3E_sU9y2Hvgy7RJzoqMZueTOII",
            "alg": "RS256",
            "n": "2oJH7dZhbIjjSDjGR69v1aC1S-mZ3LSkNgXrVpngkpZLvNJaxxDUCbrQR7nBHamOwDHBXDRq_GbU5H8ZEG8P_9TjhKHVDr6PwzahJNoXliegQlVXurtcbzrWrYoJy30fw-rWPhyjQhadLiEChtx6a9BpMek1WicfwzGXAVjQip06U8tUTN9KxMhDYIRAd0FJgu-IRhsDImHwoGP2JsqsvSndE6Dw5vc-mo8koZR_2I14Qd8zeq3mBBsRi6JRl3Y0qOjPQCFrQAPt6LnGHkCFiQmqsKozBZbeWmRZbIhA-1sHsx9Qs5TtzUaXYHz9oIpT02-rQXqRxxCaDrTl2OsImQ",
            "e": "AQAB"
        }
    ]
}`)

			jwks, err := utils.LoadKeySet(rawJWKS)
			Expect(err).ToNot(HaveOccurred())
			Expect(jwks.Keys).To(HaveLen(1))

			jwk := jwks.Key("2qAiW7sYZexfKki3E3E_sU9y2Hvgy7RJzoqMZueTOII")
			Expect(jwk).To(HaveLen(1))
			Expect(jwk[0].Algorithm).To(Equal("RS256"))
			Expect(jwk[0].KeyID).To(Equal("2qAiW7sYZexfKki3E3E_sU9y2Hvgy7RJzoqMZueTOII"))
			Expect(jwk[0].Use).To(Equal("sig"))
			Expect(jwk[0].Valid()).To(BeTrue())
			Expect(jwk[0].IsPublic()).To(BeTrue())
		})

		It("should fail to load invalid openid configuration", func() {
			rawJWKS := []byte(`invalid jwks`)

			_, err := utils.LoadKeySet(rawJWKS)
			Expect(err).To(MatchError(ContainSubstring("failed to unmarshal JWKS")))
		})

	})
})
