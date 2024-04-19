// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package dynamiccert

import (
	"crypto/tls"
	"slices"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

// DynamicCertificate implements [tls.Config.GetCertificate].
// It returns a TLS certificate and refreshes if needed.
type DynamicCertificate struct {
	certFile string
	keyFile  string

	interval    time.Duration
	certificate *tls.Certificate
	log         logr.Logger
	lock        sync.RWMutex
}

// New returns a new instance of [DynamicCertificate].
func New(certFile, keyFile string, opts ...Option) (*DynamicCertificate, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	dynamicCert := &DynamicCertificate{
		certFile:    certFile,
		keyFile:     keyFile,
		certificate: &cert,
		interval:    time.Minute,
		log:         logr.Discard(),
	}

	for _, opt := range opts {
		opt(dynamicCert)
	}

	go func() {
		ticker := time.NewTicker(dynamicCert.interval)
		for range ticker.C {
			if err := dynamicCert.reloadCert(); err != nil {
				dynamicCert.log.Error(err, "Failed to reload certificates")
			}
		}
	}()

	return dynamicCert, nil
}

func (dc *DynamicCertificate) reloadCert() error {
	cert, err := tls.LoadX509KeyPair(dc.certFile, dc.keyFile)
	if err != nil {
		return err
	}
	dc.lock.Lock()
	defer dc.lock.Unlock()
	if areEqual(cert.Certificate, dc.certificate.Certificate) {
		// do not renew the certificate if the current equals the new
		return nil
	}
	dc.certificate = &cert
	dc.log.Info("Certificate was reloaded")
	return nil
}

func areEqual(cert1 [][]byte, cert2 [][]byte) bool {
	if len(cert1) != len(cert2) {
		return false
	}

	for i := range cert1 {
		if !slices.Equal(cert1[i], cert2[i]) {
			return false
		}
	}
	return true
}

// GetCertificate returns the current loaded certificate.
func (dc *DynamicCertificate) GetCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	dc.lock.RLock()
	defer dc.lock.RUnlock()
	return dc.certificate, nil
}

// Option can be used to configure [DynamicCertificate].
type Option func(*DynamicCertificate)

// WithRefreshInterval sets the interval that will be used
// to periodically check in the TLS certificate should be refreshed.
func WithRefreshInterval(interval time.Duration) Option {
	return func(dc *DynamicCertificate) {
		dc.interval = interval
	}
}

// WithLogger sets the logger for [DynamicCertificate].
func WithLogger(log logr.Logger) Option {
	return func(dc *DynamicCertificate) {
		dc.log = log
	}
}
