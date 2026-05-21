// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Test Suite")
}

const (
	discoveryServerBaseURI = "https://discovery.local.gardener.cloud"
)

var (
	parentCtx                     = context.Background()
	discoveryClient               *http.Client
	gardenRuntimeClusterClientset *kubernetes.Clientset
)

var _ = BeforeEach(func() {
	parentCtx = context.Background()
})

var _ = BeforeSuite(func() {
	kubeconfigPath := os.Getenv("KUBECONFIG_RUNTIME_CLUSTER")
	kubeconfigBytes, err := os.ReadFile(kubeconfigPath)
	Expect(err).ToNot(HaveOccurred())

	cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	Expect(err).ToNot(HaveOccurred())
	gardenRuntimeClusterClientset, err = kubernetes.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())

	secrets, err := gardenRuntimeClusterClientset.CoreV1().Secrets("garden").List(parentCtx, metav1.ListOptions{LabelSelector: "name=gardener-discovery-server-tls,manager-identity=gardener-operator"})
	Expect(err).ToNot(HaveOccurred())
	Expect(secrets.Items).ToNot(BeEmpty())
	pool := x509.NewCertPool()
	for _, secret := range secrets.Items {
		pool.AppendCertsFromPEM(secret.Data["tls.crt"])
	}

	discoveryClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
			},
		},
	}
})
