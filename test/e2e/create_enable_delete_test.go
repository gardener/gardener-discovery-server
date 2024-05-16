// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"time"

	"github.com/go-jose/go-jose/v4"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Managed Issuer Tests", Label("ManagedIssuer"), func() {
	f := defaultShootCreationFramework()
	f.Shoot = defaultShoot("e2e-default")

	It("Create Shoot, Enable Managed Issuer, Delete Shoot", Label("good-case"), func() {
		By("Create Shoot")
		ctx, cancel := context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
		f.Verify()

		resp, err := getWellKnownForShoot(parentCtx, f.Shoot.ObjectMeta.UID)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		resp.Body.Close()

		resp, err = getJWKSForShoot(parentCtx, f.Shoot.ObjectMeta.UID)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		resp.Body.Close()

		By("Enable Managed Issuer")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, f.Shoot, addAnnotations)).To(Succeed())

		By("Check that the Discovery Server is able to serve the shoot's OIDC discovery documents")

		configSecret, err := gardenClusterClientset.CoreV1().Secrets("garden").Get(parentCtx, "shoot-service-account-issuer", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		hostname := string(configSecret.Data["hostname"])
		resp, err = getWellKnownForShoot(parentCtx, f.Shoot.ObjectMeta.UID)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		wellKnownBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		Expect(err).ToNot(HaveOccurred())
		var wellKnown map[string]any
		Expect(json.Unmarshal(wellKnownBytes, &wellKnown)).To(Succeed())
		iss, ok := wellKnown["issuer"].(string)
		Expect(ok).To(BeTrue())
		Expect(iss).To(Equal("https://" + hostname + "/projects/local/shoots/" + string(f.Shoot.ObjectMeta.UID) + "/issuer"))
		jwksURI, ok := wellKnown["jwks_uri"].(string)
		Expect(ok).To(BeTrue())
		Expect(jwksURI).To(Equal("https://" + hostname + "/projects/local/shoots/" + string(f.Shoot.ObjectMeta.UID) + "/issuer/jwks"))

		resp, err = getJWKSForShoot(parentCtx, f.Shoot.ObjectMeta.UID)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		jwks, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		Expect(err).ToNot(HaveOccurred())

		keySet := jose.JSONWebKeySet{}
		Expect(json.Unmarshal(jwks, &keySet)).To(Succeed())
		Expect(keySet.Keys).To(HaveLen(1))
		pubKey, ok := (keySet.Keys[0].Key).(*rsa.PublicKey)
		Expect(ok).To(BeTrue())

		_, seedClient, err := f.GetSeed(ctx, *f.Shoot.Status.SeedName)
		Expect(err).ToNot(HaveOccurred())
		project, err := f.GetShootProject(ctx, f.Shoot.Namespace)
		Expect(err).ToNot(HaveOccurred())
		shootSeedNamespace := framework.ComputeTechnicalID(project.Name, f.Shoot)

		By("Check that the received public key is indeed the same public key that resides in the seed")
		secretList := &corev1.SecretList{}
		Expect(seedClient.Client().List(ctx, secretList, client.InNamespace(shootSeedNamespace), client.MatchingLabels{"bundle-for": "service-account-key"})).To(Succeed())
		Expect(secretList.Items).To(HaveLen(1))

		keyBundleBlock, _ := pem.Decode(secretList.Items[0].Data["bundle.key"])
		rsaKey, err := x509.ParsePKCS1PrivateKey(keyBundleBlock.Bytes)
		Expect(err).ToNot(HaveOccurred())
		Expect(rsaKey.PublicKey.Equal(pubKey)).To(BeTrue())

		By("Delete Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.DeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())

		resp, err = getWellKnownForShoot(parentCtx, f.Shoot.ObjectMeta.UID)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		resp.Body.Close()

		resp, err = getJWKSForShoot(parentCtx, f.Shoot.ObjectMeta.UID)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		resp.Body.Close()
	})
})
