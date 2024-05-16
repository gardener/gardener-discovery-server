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
	"os/exec"
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
	discoveryServerBaseURI = "https://localhost:10443"
)

var (
	cmd                    *exec.Cmd
	parentCtx              = context.Background()
	discoveryClient        *http.Client
	gardenClusterClientset *kubernetes.Clientset
)

var _ = BeforeEach(func() {
	parentCtx = context.Background()
})

var _ = BeforeSuite(func() {
	// let's hope that this is stable and does not introduces flakiness
	// port-forward can be revisited later on if it causes unstable connection during tests
	cmd = exec.Command("kubectl", "-n", "garden", "port-forward", "service/gardener-discovery-server", "10443:10443")
	Expect(cmd.Start()).To(Succeed())

	kubeconfigPath := os.Getenv("KUBECONFIG")
	kubeconfigBytes, err := os.ReadFile(kubeconfigPath)
	Expect(err).ToNot(HaveOccurred())

	cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	Expect(err).ToNot(HaveOccurred())
	gardenClusterClientset, err = kubernetes.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())

	secret, err := gardenClusterClientset.CoreV1().Secrets("garden").Get(parentCtx, "gardener-discovery-server-tls", metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(secret.Data["tls.crt"])
	discoveryClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
			},
		},
	}

})

var _ = AfterSuite(func() {
	Expect(cmd.Process.Kill()).To(Succeed())
})
