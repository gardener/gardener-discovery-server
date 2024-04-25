// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package dynamiccert_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDynamicCert(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dynamic Certificate Test Suite")
}

const (
	// test data directory
	testdataDir = "./testdata"
)

var (
	serverkey  = filepath.Join(testdataDir, "tls.key")
	servercert = filepath.Join(testdataDir, "tls.crt")
)

var _ = BeforeSuite(func() {
	Expect(generateTestData()).To(Succeed())
})

var _ = AfterSuite(func() {
	Expect(os.RemoveAll(testdataDir)).To(Succeed())
})

func generateTestData() error {
	if _, err := os.Stat(testdataDir); !os.IsNotExist(err) {
		// if directory already exists, remove it so new certificates are generated
		if err := os.RemoveAll(testdataDir); err != nil {
			return err
		}
	}
	err := os.Mkdir(testdataDir, 0750)
	if err != nil {
		return err
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}

	now := time.Now()
	cert := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		DNSNames: []string{
			"localhost",
		},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:             now,
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &cert, &cert, &key.PublicKey, key)
	if err != nil {
		return err
	}

	certFile, err := os.OpenFile(filepath.Clean(servercert), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	if err := pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}); err != nil {
		return err
	}

	if err := certFile.Close(); err != nil {
		return err
	}

	keyFile, err := os.OpenFile(filepath.Clean(serverkey), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}

	if err := pem.Encode(keyFile, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	}); err != nil {
		return err
	}

	return keyFile.Close()
}
