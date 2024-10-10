// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package workloadidentity_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"k8s.io/utils/ptr"
	logzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/gardener/gardener-discovery-server/internal/handler/workloadidentity"
	"github.com/gardener/gardener-discovery-server/internal/utils"
)

var _ = Describe("#WorkloadIdentity", func() {
	const pathPrefix = "/garden/workload-identity/issuer"
	var (
		openIDConfig []byte
		jwks         []byte
		handler      *workloadidentity.Handler
		mux          *http.ServeMux
		logger       logr.Logger

		headers map[string]string
	)

	BeforeEach(func() {
		logger = logzap.New(logzap.WriteTo(GinkgoWriter))
		var (
			iss = "https://test.discovery-server.gardener.cloud.local" + pathPrefix
			err error
		)

		openIDConfig, err = createOpenIDMeta(iss, iss+"/jwks")
		Expect(err).ToNot(HaveOccurred())

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		Expect(err).ToNot(HaveOccurred())

		publicKey := privateKey.Public()
		kid, err := getKeyID(publicKey)
		Expect(err).ToNot(HaveOccurred())

		jwks, err = createJWKS(publicKey, kid)
		Expect(err).ToNot(HaveOccurred())

		handler, err = workloadidentity.New(openIDConfig, jwks, logger)
		Expect(err).ToNot(HaveOccurred())

		mux = http.NewServeMux()
		mux.Handle(pathPrefix+"/.well-known/openid-configuration", handler.HandleOpenIDConfiguration())
		mux.Handle(pathPrefix+"/jwks", handler.HandleJWKS())

		headers = map[string]string{
			"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
			"Content-Type":              "application/json",
			"Cache-Control":             "public, max-age=3600",
		}
	})

	Describe("#New", func() {
		DescribeTable("Issuer url",
			func(iss string, matcher types.GomegaMatcher) {
				openIDConfig, err := createOpenIDMeta(iss, iss+"/jwks")
				Expect(err).ToNot(HaveOccurred())

				_, err = workloadidentity.New(openIDConfig, jwks, logger)
				Expect(err).To(matcher)
			},
			Entry("should not allow issuer url with control characters", "https://foo.\n.bar", MatchError(ContainSubstring("failed to parse issuer url"))),
			Entry("should not allow issuer url using scheme other than https", "ftp://foo.bar", MatchError("invalid issuer url scheme")),
			Entry("should not allow issuer url using query", "https://foo.bar/?baz=42", MatchError("issuer url must not contain query")),
			Entry("should not allow issuer url using fragment", "https://foo.bar/#baz", MatchError("issuer url must not contain fragment")),
			Entry("should allow valid issuer url", "https://foo.bar/issuer", Not(HaveOccurred())),
		)

		DescribeTable("JWKS URL",
			func(jwkURL string, matcher types.GomegaMatcher) {
				openIDConfig, err := createOpenIDMeta("https://foo.bar", jwkURL)
				Expect(err).ToNot(HaveOccurred())

				_, err = workloadidentity.New(openIDConfig, jwks, logger)
				Expect(err).To(matcher)
			},
			Entry("should not allow jwks url with control characters", "https://foo.\n.bar/jwks", MatchError(ContainSubstring("failed to parse jwks url"))),
			Entry("should not allow jwks url using scheme other than https", "ftp://foo.bar/jwks", MatchError("invalid jwks url scheme")),
			Entry("should allow valid jwks url", "https://foo.bar/jwks", Not(HaveOccurred())),
		)

		It("should fail to create new handler when the there is a non-public JWK in the key set", func() {
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).ToNot(HaveOccurred())

			kid := "id-test"
			jwks, err := createJWKS(privateKey, kid)
			Expect(err).ToNot(HaveOccurred())

			_, err = workloadidentity.New(openIDConfig, jwks, logger)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Errorf("jwks key with id %q is not public", kid)))
		})

		It("should fail to load openid configuration", func() {
			_, err := workloadidentity.New([]byte(`invalid openid configuration}`), jwks, logger)
			Expect(err).To(MatchError(ContainSubstring("failed to load openid configuration")))
		})

		It("should fail to load json web key set", func() {
			_, err := workloadidentity.New(openIDConfig, []byte(`invalid json web key set}`), logger)
			Expect(err).To(MatchError(ContainSubstring("failed to load json web key set")))
		})
	})

	DescribeTable("#handleRequest",
		func(method, path string, expectedStatus int, expectedResponse *[]byte, expectedHeaders map[string]string) {
			request := httptest.NewRequest(method, path, nil)
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, request)

			Expect(recorder).To(HaveHTTPStatus(expectedStatus))
			Expect(recorder).To(HaveHTTPBody(string(*expectedResponse)))
			for k, v := range expectedHeaders {
				Expect(recorder).To(HaveHTTPHeaderWithValue(k, v))
			}
		},
		Entry("[OpenIDConfiguration] it should successfully get document",
			http.MethodGet, pathPrefix+"/.well-known/openid-configuration", http.StatusOK, &openIDConfig, headers),
		Entry("[OpenIDConfiguration] it should successfully head document",
			http.MethodHead, pathPrefix+"/.well-known/openid-configuration", http.StatusOK, &openIDConfig, headers),
		Entry("[OpenIDConfiguration] it should fail post document",
			http.MethodPost, pathPrefix+"/.well-known/openid-configuration", http.StatusMethodNotAllowed, ptr.To([]byte(`{"code":405,"message":"method not allowed"}`)), headers),
		Entry("[JWKS] it should successfully get document",
			http.MethodGet, pathPrefix+"/jwks", http.StatusOK, &jwks, headers),
		Entry("[JWKS] it should successfully head document",
			http.MethodHead, pathPrefix+"/jwks", http.StatusOK, &jwks, headers),
		Entry("[JWKS] it should fail post document",
			http.MethodPost, pathPrefix+"/jwks", http.StatusMethodNotAllowed, ptr.To([]byte(`{"code":405,"message":"method not allowed"}`)), headers),
		Entry("it should return not found on other paths",
			http.MethodGet, pathPrefix, http.StatusNotFound, ptr.To([]byte("404 page not found\n")), nil),
	)
})

func getKeyID(publicKey crypto.PublicKey) (string, error) {
	marshaled, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}
	shaSum := sha256.Sum256(marshaled)
	return base64.RawURLEncoding.EncodeToString(shaSum[:]), nil
}

func createOpenIDMeta(iss, jwks string) ([]byte, error) {
	openIDMetadata := utils.OpenIDMetadata{
		Issuer:  iss,
		JWKSURI: jwks,
	}
	return json.Marshal(openIDMetadata)
}

func createJWKS(key any, kid string) ([]byte, error) {
	keySet := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{{
			Algorithm: string(jose.RS256),
			Key:       key,
			Use:       "sig",
			KeyID:     kid,
		}},
	}
	return json.Marshal(keySet)
}
