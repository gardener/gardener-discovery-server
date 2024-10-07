// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"net/http"
	"sync"

	"github.com/go-logr/logr"
)

const (
	headerContentType = "Content-Type"
	mimeAppJSON       = "application/json"
)

// SetHSTS is middleware handler setting Strict-Transport-Security header.
func SetHSTS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if w.Header().Get("Strict-Transport-Security") == "" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// AllowMethods is middleware handler restricting the allowed http methods.
func AllowMethods(next http.Handler, log logr.Logger, allowedMethods ...string) http.Handler {
	var (
		responseMethodNotAllowed = []byte(`{"code":405,"message":"method not allowed"}`)
		methods                  = sync.Map{}
	)
	for _, m := range allowedMethods {
		methods.Store(m, nil)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := methods.Load(r.Method); !ok {
			w.Header().Set(headerContentType, mimeAppJSON)
			w.WriteHeader(http.StatusMethodNotAllowed)
			if _, err := w.Write(responseMethodNotAllowed); err != nil {
				log.Error(err, "Failed writing response")
				return
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}

// NotFound is handler replying with not found.
func NotFound(log logr.Logger) http.Handler {
	var responseNotFound = []byte(`{"code":404,"message":"not found"}`)

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write(responseNotFound); err != nil {
			log.Error(err, "Failed writing not found response")
			return
		}
	})
}
