// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	logzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/gardener/gardener-discovery-server/internal/handler"
	"github.com/gardener/gardener-discovery-server/internal/store"
)

var _ = Describe("#Handler", func() {
	var (
		noOpHandler http.Handler
		log         logr.Logger
	)

	BeforeEach(func() {
		noOpHandler = http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})
		log = logzap.New(logzap.WriteTo(GinkgoWriter))
	})

	Describe("#SetHSTS", func() {
		It("should set only HSTS headers", func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			resp := httptest.NewRecorder()

			h := handler.SetHSTS(noOpHandler)

			h.ServeHTTP(resp, req)
			Expect(resp).To(HaveHTTPHeaderWithValue("Strict-Transport-Security", "max-age=31536000; includeSubDomains"))
			Expect(resp.Header()).To(HaveLen(1))
		})

		It("should not overwrite already set HSTS headers", func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			resp := httptest.NewRecorder()

			resp.Header().Set("Strict-Transport-Security", "max-age=1")

			h := handler.SetHSTS(noOpHandler)

			h.ServeHTTP(resp, req)
			Expect(resp).To(HaveHTTPHeaderWithValue("Strict-Transport-Security", "max-age=1"))
			Expect(resp.Header()).To(HaveLen(1))
		})
	})

	Describe("#AllowMethods", func() {
		It("should allow allowed methods", func() {
			allowedMethods := []string{http.MethodGet, http.MethodConnect}
			req := httptest.NewRequest(allowedMethods[0], "/", nil)
			resp := httptest.NewRecorder()

			h := handler.AllowMethods(noOpHandler, log, allowedMethods...)

			h.ServeHTTP(resp, req)
			Expect(resp).To(HaveHTTPStatus(http.StatusOK))
			Expect(resp).To(HaveHTTPBody(""))
			Expect(resp.Header()).To(HaveLen(0))
		})

		It("should disallow not allowed method", func() {
			allowedMethods := []string{http.MethodGet, http.MethodPut}
			disallowedMethod := http.MethodDelete
			req := httptest.NewRequest(disallowedMethod, "/", nil)
			resp := httptest.NewRecorder()

			h := handler.AllowMethods(noOpHandler, log, allowedMethods...)

			h.ServeHTTP(resp, req)
			Expect(resp).To(HaveHTTPStatus(http.StatusMethodNotAllowed))
			Expect(resp).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			Expect(resp).To(HaveHTTPBody(`{"code":405,"message":"method not allowed"}`))
		})
	})

	Describe("#NotFound", func() {
		It("should return not found", func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			resp := httptest.NewRecorder()

			h := handler.NotFound(log)

			h.ServeHTTP(resp, req)
			Expect(resp).To(HaveHTTPStatus(http.StatusNotFound))
			Expect(resp).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			Expect(resp).To(HaveHTTPBody(`{"code":404,"message":"not found"}`))
		})
	})

	Describe("#StoreRequest", func() {
		var s *store.Store[string]

		BeforeEach(func() {
			var err error
			s, err = store.NewStore(func(s string) string {
				return s
			})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return the data written to store", func() {
			id := uuid.NewString()
			s.Write("test--"+id, "entry")

			req := httptest.NewRequest(http.MethodGet, "/projects/test/shoots/"+id+"/test", nil)
			req.Pattern = "/projects/{projectName}/shoots/{shootUID}/test"
			req.SetPathValue("projectName", "test")
			req.SetPathValue("shootUID", id)

			resp := httptest.NewRecorder()

			h := handler.StoreRequest(log, s, func(data string) []byte { return []byte(`{"data":"` + data + `"}`) })
			h.ServeHTTP(resp, req)

			Expect(resp).To(HaveHTTPStatus(http.StatusOK))
			Expect(resp).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			Expect(resp).To(HaveHTTPHeaderWithValue("Cache-Control", "public, max-age=3600"))
			Expect(resp).To(HaveHTTPBody(`{"data":"entry"}`))
		})

		It("should return not found if data is not in store", func() {
			id := uuid.NewString()
			req := httptest.NewRequest(http.MethodGet, "/projects/test/shoots/"+id+"/test", nil)
			req.Pattern = "/projects/{projectName}/shoots/{shootUID}/test"
			req.SetPathValue("projectName", "test")
			req.SetPathValue("shootUID", id)

			resp := httptest.NewRecorder()

			h := handler.StoreRequest(log, s, func(data string) []byte { return []byte(`{"data":"` + data + `"}`) })
			h.ServeHTTP(resp, req)

			Expect(resp).To(HaveHTTPStatus(http.StatusNotFound))
			Expect(resp).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			Expect(resp).To(HaveHTTPBody(`{"code":404,"message":"not found"}`))
		})

		It("should return bad request if path value shootUID is invalid", func() {
			id := uuid.NewString()
			req := httptest.NewRequest(http.MethodGet, "/projects/test/shoots/"+id+"/test", nil)
			req.Pattern = "/projects/{projectName}/shoots/{shootUID}/test"
			req.SetPathValue("projectName", "test")
			req.SetPathValue("shootUID", "invalid")

			resp := httptest.NewRecorder()

			h := handler.StoreRequest(log, s, func(data string) []byte { return []byte(`{"data":"` + data + `"}`) })
			h.ServeHTTP(resp, req)

			Expect(resp).To(HaveHTTPStatus(http.StatusBadRequest))
			Expect(resp).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			Expect(resp).To(HaveHTTPBody(`{"code":400,"message":"invalid UID"}`))
		})
	})
})
