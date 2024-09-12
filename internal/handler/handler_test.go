// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	logzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/gardener/gardener-discovery-server/internal/handler"
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
			Expect(resp).To(HaveHTTPHeaderWithValue("Strict-Transport-Security", "max-age=31536000"))
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

	Describe("#HandleNotFound", func() {
		It("should return not found", func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			resp := httptest.NewRecorder()

			h := handler.HandleNotFound(log)

			h.ServeHTTP(resp, req)
			Expect(resp).To(HaveHTTPStatus(http.StatusNotFound))
			Expect(resp).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			Expect(resp).To(HaveHTTPBody(`{"code":404,"message":"not found"}`))
			Expect(resp).To(HaveHTTPHeaderWithValue("Strict-Transport-Security", "max-age=31536000"))
		})
	})
})
