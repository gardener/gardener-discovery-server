// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package certificate_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	logzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/gardener/gardener-discovery-server/internal/handler"
	certhandler "github.com/gardener/gardener-discovery-server/internal/handler/certificate"
	"github.com/gardener/gardener-discovery-server/internal/store"
	certstore "github.com/gardener/gardener-discovery-server/internal/store/certificate"
)

var _ = Describe("#HttpHandlerCertificate", func() {
	var (
		s *store.Store[certstore.Data]

		projectName = "foo"

		uid1 = "a6475c90-d533-43c4-bbb0-d99200b491b1"
		uid2 = "1e4914ca-c837-451d-a1cf-c559d131cb57"

		certHandler *certhandler.Handler
		mux         *http.ServeMux
	)

	BeforeEach(func() {
		s = store.MustNewStore(certstore.Copy)
		s.Write(projectName+"--"+uid1, certstore.Data{
			CABundle: []byte("bundle1"),
		})
		s.Write(projectName+"--"+uid2, certstore.Data{
			CABundle: []byte("bundle2"),
		})

		log := logzap.New(logzap.WriteTo(GinkgoWriter))
		certHandler = certhandler.New(s, log)
		mux = http.NewServeMux()
		mux.Handle("/projects/{projectName}/shoots/{shootUID}/cluster-ca", certHandler.HandleCABundle())
		mux.Handle("/", handler.NotFound(log))
	})

	DescribeTable(
		"requests",
		func(method string, uri string, expectedStatus int, expectedResponseBytes []byte, expectedHeaders map[string]string) {
			req := httptest.NewRequest(method, uri, nil)
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
			http.MethodGet,
			"https://abc.def/projects/foo/shoots/a6475c90-d533-43c4-bbb0-d99200b491b1/cluster-ca",
			200,
			[]byte("bundle1"),
			map[string]string{
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"Content-Type":              "application/json",
				"Cache-Control":             "public, max-age=3600",
			},
		),
		Entry(
			"should return config for uid2",
			http.MethodGet,
			"https://abc.def/projects/foo/shoots/1e4914ca-c837-451d-a1cf-c559d131cb57/cluster-ca",
			200,
			[]byte("bundle2"),
			map[string]string{
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"Content-Type":              "application/json",
				"Cache-Control":             "public, max-age=3600",
			},
		),
		Entry(
			"should return not found",
			http.MethodGet,
			"https://abc.def/projects/not-existent/shoots/1e4914ca-c837-451d-a1cf-c559d131cb57/cluster-ca",
			404,
			[]byte(`{"code":404,"message":"not found"}`),
			map[string]string{
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"Content-Type":              "application/json",
			},
		),
		Entry(
			"should return bad request",
			http.MethodGet,
			"https://abc.def/projects/not-existent/shoots/not-a-uuid/cluster-ca",
			400,
			[]byte(`{"code":400,"message":"bad request"}`),
			map[string]string{
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"Content-Type":              "application/json",
			},
		),
		Entry(
			"should return not found when querying a non existent endpoint",
			http.MethodGet,
			"https://abc.def/does-not-exist",
			404,
			[]byte(`{"code":404,"message":"not found"}`),
			map[string]string{
				"Content-Type": "application/json",
			},
		),
		Entry(
			"should return method not allowed when querying the ca endpoint with POST",
			http.MethodPost,
			"https://abc.def/projects/foo/shoots/1e4914ca-c837-451d-a1cf-c559d131cb57/cluster-ca",
			405,
			[]byte(`{"code":405,"message":"method not allowed"}`),
			map[string]string{
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"Content-Type":              "application/json",
			},
		),
	)
})
