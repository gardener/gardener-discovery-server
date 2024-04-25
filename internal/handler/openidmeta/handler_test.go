// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openidmeta_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	logzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	oidhandler "github.com/gardener/gardener-discovery-server/internal/handler/openidmeta"
	oidstore "github.com/gardener/gardener-discovery-server/internal/store/openidmeta"
)

var _ = Describe("#HttpHandlerOpenIDMeta", func() {
	var (
		store *oidstore.Store

		projectName = "foo"

		uid1 = "a6475c90-d533-43c4-bbb0-d99200b491b1"
		uid2 = "1e4914ca-c837-451d-a1cf-c559d131cb57"

		handler *oidhandler.Handler
		mux     *http.ServeMux
	)

	BeforeEach(func() {
		store = oidstore.NewStore()
		store.Write(projectName+"--"+uid1, oidstore.Data{
			Config: []byte("config1"),
			JWKS:   []byte("jwks1"),
		})
		store.Write(projectName+"--"+uid2, oidstore.Data{
			Config: []byte("config2"),
			JWKS:   []byte("jwks2"),
		})

		handler = oidhandler.New(store, logzap.New(logzap.WriteTo(GinkgoWriter)))
		mux = http.NewServeMux()
		mux.HandleFunc("GET /projects/{projectName}/shoots/{shootUID}/issuer/.well-known/openid-configuration", handler.HandleWellKnown)
		mux.HandleFunc("GET /projects/{projectName}/shoots/{shootUID}/issuer/jwks", handler.HandleJWKS)
	})

	DescribeTable(
		"requests",
		func(uri string, expectedStatus int, expectedResponseBytes []byte, expectedHeaders map[string]string) {
			req := httptest.NewRequest("GET", uri, nil)
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(expectedStatus))
			Expect(recorder.Body.Bytes()).To(Equal(expectedResponseBytes))
			Expect(len(recorder.Result().Header)).To(Equal(len(expectedHeaders)))
			for k, v := range expectedHeaders {
				Expect(recorder.Result().Header[k]).To(Equal([]string{v}))
			}
		},
		Entry(
			"should return config for uid1",
			"https://abc.def/projects/foo/shoots/a6475c90-d533-43c4-bbb0-d99200b491b1/issuer/.well-known/openid-configuration",
			200,
			[]byte("config1"),
			map[string]string{
				"Content-Type":  "application/json",
				"Cache-Control": "public, max-age=3600",
			},
		),
		Entry(
			"should return jwks for uid1",
			"https://abc.def/projects/foo/shoots/a6475c90-d533-43c4-bbb0-d99200b491b1/issuer/jwks",
			200,
			[]byte("jwks1"),
			map[string]string{
				"Content-Type":  "application/json",
				"Cache-Control": "public, max-age=3600",
			},
		),
		Entry(
			"should return config for uid2",
			"https://abc.def/projects/foo/shoots/1e4914ca-c837-451d-a1cf-c559d131cb57/issuer/.well-known/openid-configuration",
			200,
			[]byte("config2"),
			map[string]string{
				"Content-Type":  "application/json",
				"Cache-Control": "public, max-age=3600",
			},
		),
		Entry(
			"should return jwks for uid2",
			"https://abc.def/projects/foo/shoots/1e4914ca-c837-451d-a1cf-c559d131cb57/issuer/jwks",
			200,
			[]byte("jwks2"),
			map[string]string{
				"Content-Type":  "application/json",
				"Cache-Control": "public, max-age=3600",
			},
		),
		Entry(
			"should return not found querying the config endpoint",
			"https://abc.def/projects/not-existent/shoots/1e4914ca-c837-451d-a1cf-c559d131cb57/issuer/.well-known/openid-configuration",
			404,
			[]byte(`{"code":404,"message":"not found"}`),
			map[string]string{
				"Content-Type": "application/json",
			},
		),
		Entry(
			"should return not found querying the jwks endpoint",
			"https://abc.def/projects/not-existent/shoots/1e4914ca-c837-451d-a1cf-c559d131cb57/issuer/jwks",
			404,
			[]byte(`{"code":404,"message":"not found"}`),
			map[string]string{
				"Content-Type": "application/json",
			},
		),
		Entry(
			"should return bad request querying the config endpoint",
			"https://abc.def/projects/not-existent/shoots/not-a-uuid/issuer/.well-known/openid-configuration",
			400,
			[]byte(`{"code":400,"message":"bad request"}`),
			map[string]string{
				"Content-Type": "application/json",
			},
		),
		Entry(
			"should return bad request querying the jwks endpoint",
			"https://abc.def/projects/not-existent/shoots/not-a-uuid/issuer/jwks",
			400,
			[]byte(`{"code":400,"message":"bad request"}`),
			map[string]string{
				"Content-Type": "application/json",
			},
		),
	)
})
