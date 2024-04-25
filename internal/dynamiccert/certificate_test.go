// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package dynamiccert_test

import (
	"crypto/tls"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	logzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/gardener/gardener-discovery-server/internal/dynamiccert"
)

var _ = Describe("#DynamicCertificate", func() {
	var (
		dynCert *dynamiccert.DynamicCertificate
	)
	BeforeEach(func() {
		var err error
		dynCert, err = dynamiccert.New(
			servercert,
			serverkey,
			dynamiccert.WithRefreshInterval(time.Millisecond*100),
			dynamiccert.WithLogger(logzap.New(logzap.WriteTo(GinkgoWriter))),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should consistently return the same certificate", func() {
		cert, err := dynCert.GetCertificate(&tls.ClientHelloInfo{})
		Expect(err).ToNot(HaveOccurred())
		Expect(cert).ToNot(BeNil())
		Expect(cert.Certificate).To(HaveLen(1))

		Consistently(func(g Gomega) {
			gotCert, err := dynCert.GetCertificate(&tls.ClientHelloInfo{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(gotCert).ToNot(BeNil())
			g.Expect(gotCert.Certificate).To(HaveLen(1))
			g.Expect(cert.Certificate[0]).To(Equal(gotCert.Certificate[0]))
		}, "400ms", "100ms").Should(Succeed())
	})

	It("should eventually return the new certificate", func() {
		cert, err := dynCert.GetCertificate(&tls.ClientHelloInfo{})
		Expect(err).ToNot(HaveOccurred())
		Expect(cert).ToNot(BeNil())
		Expect(cert.Certificate).To(HaveLen(1))

		go func() {
			defer GinkgoRecover()
			// regenerate certificates
			Expect(generateTestData()).To(Succeed())
		}()

		Eventually(func(g Gomega) {
			gotCert, err := dynCert.GetCertificate(&tls.ClientHelloInfo{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(gotCert).ToNot(BeNil())
			g.Expect(gotCert.Certificate).To(HaveLen(1))
			g.Expect(cert.Certificate[0]).NotTo(Equal(gotCert.Certificate[0]))
		}, "400ms", "100ms").Should(Succeed())
	})
})
