// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package certificate_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	logzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	certreconciler "github.com/gardener/gardener-discovery-server/internal/reconciler/certificate"
	"github.com/gardener/gardener-discovery-server/internal/store"
	certstore "github.com/gardener/gardener-discovery-server/internal/store/certificate"
)

var _ = Describe("#ReconcileCertificate", func() {
	const (
		projectName    = "abc"
		shootName      = "my-shoot"
		shootNamespace = "garden-" + projectName
		resyncPeriod   = time.Second
	)

	var (
		reconciler *certreconciler.Reconciler

		c client.Client
		s *store.Store[certstore.Data]

		namespace               *corev1.Namespace
		shoot                   *gardencorev1beta1.Shoot
		project                 *gardencorev1beta1.Project
		configmap               *corev1.ConfigMap
		configmapNamespacedName types.NamespacedName

		ctx      = logf.IntoContext(context.Background(), logzap.New(logzap.WriteTo(GinkgoWriter)))
		shootUID = types.UID("7a25a9b8-f7fc-4e1e-a421-31b4deaa3086")
		storeKey = projectName + "--" + string(shootUID)

		expectedBundleBytes []byte

		expectStoreEntry = func(store *store.Store[certstore.Data], key string, want certstore.Data) {
			got, ok := store.Read(key)
			Expect(ok).To(BeTrue())
			Expect(got).To(Equal(want))
		}

		generateCA = func() ([]byte, error) {
			serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 10000)
			serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
			if err != nil {
				return nil, err
			}

			cert := &x509.Certificate{
				SerialNumber: serialNumber,
				Subject: pkix.Name{
					Organization: []string{"Test"},
					Country:      []string{"Test"},
				},
				NotBefore:             time.Now(),
				NotAfter:              time.Now().AddDate(0, 0, 3),
				IsCA:                  true,
				ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
				KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
				BasicConstraintsValid: true,
			}

			key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				return nil, err
			}

			certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
			if err != nil {
				return nil, err
			}

			pemEncoded := &bytes.Buffer{}
			err = pem.Encode(pemEncoded, &pem.Block{
				Type:  "CERTIFICATE",
				Bytes: certBytes,
			})
			Expect(err).ToNot(HaveOccurred())

			return pemEncoded.Bytes(), nil
		}
	)

	BeforeEach(func() {
		c = fake.NewClientBuilder().WithScheme(kubernetes.GardenScheme).Build()
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: shootNamespace,
				Labels: map[string]string{
					"project.gardener.cloud/name": projectName,
				},
			},
		}

		shoot = &gardencorev1beta1.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      shootName,
				Namespace: shootNamespace,
				UID:       shootUID,
			},
		}
		project = &gardencorev1beta1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: projectName,
			},
			Spec: gardencorev1beta1.ProjectSpec{
				Namespace: ptr.To(shootNamespace),
			},
		}

		ca, err := generateCA()
		Expect(err).ToNot(HaveOccurred())

		bundle := struct {
			Certs string `json:"certs"`
		}{
			Certs: string(ca),
		}

		expectedBundleBytes, err = json.Marshal(bundle)
		Expect(err).ToNot(HaveOccurred())

		configmap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shoot-ca",
				Namespace: shootNamespace,
				Labels: map[string]string{
					"discovery.gardener.cloud/public":   "shoot-ca",
					"gardener.cloud/update-restriction": "true",
					"gardener.cloud/project":            project.Name,
					"shoot.gardener.cloud/name":         shoot.Name,
					"shoot.gardener.cloud/uid":          string(shoot.UID),
				},
			},
			Data: map[string]string{
				"ca.crt": string(ca),
			},
		}
		configmapNamespacedName = types.NamespacedName{
			Name:      configmap.Name,
			Namespace: configmap.Namespace,
		}

		s = store.MustNewStore(certstore.Copy)

		reconciler = &certreconciler.Reconciler{
			ResyncPeriod: resyncPeriod,
			Client:       c,
			Store:        s,
		}

	})

	It("should write entry to store", func() {
		Expect(c.Create(ctx, namespace)).To(Succeed())
		Expect(c.Create(ctx, project)).To(Succeed())
		Expect(c.Create(ctx, shoot)).To(Succeed())
		Expect(c.Create(ctx, configmap)).To(Succeed())

		res, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: configmapNamespacedName})
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(ctrl.Result{RequeueAfter: resyncPeriod}))

		Expect(s.Len()).To(Equal(1))
		expectStoreEntry(s, storeKey, certstore.Data{
			CABundle: expectedBundleBytes,
		})
	})

	It("should write double certificate entry to store", func() {
		Expect(c.Create(ctx, namespace)).To(Succeed())
		Expect(c.Create(ctx, project)).To(Succeed())
		Expect(c.Create(ctx, shoot)).To(Succeed())

		first, err := generateCA()
		Expect(err).ToNot(HaveOccurred())

		second, err := generateCA()
		Expect(err).ToNot(HaveOccurred())

		bundle := struct {
			Certs string `json:"certs"`
		}{
			Certs: string(first) + string(second),
		}
		configmap.Data["ca.crt"] = string(first) + string(second)

		expectedBundleBytes, err = json.Marshal(bundle)
		Expect(err).ToNot(HaveOccurred())

		Expect(c.Create(ctx, configmap)).To(Succeed())

		res, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: configmapNamespacedName})
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(ctrl.Result{RequeueAfter: resyncPeriod}))

		Expect(s.Len()).To(Equal(1))
		expectStoreEntry(s, storeKey, certstore.Data{
			CABundle: expectedBundleBytes,
		})
	})

	DescribeTable(
		"should remove entry from store because of failed validation",
		func(prepFunc func()) {
			Expect(c.Create(ctx, namespace)).To(Succeed())
			Expect(c.Create(ctx, project)).To(Succeed())
			Expect(c.Create(ctx, shoot)).To(Succeed())
			Expect(c.Create(ctx, configmap)).To(Succeed())

			res, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: configmapNamespacedName})
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ctrl.Result{RequeueAfter: resyncPeriod}))

			Expect(s.Len()).To(Equal(1))
			expectStoreEntry(s, storeKey, certstore.Data{
				CABundle: expectedBundleBytes,
			})

			prepFunc()

			res, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: configmapNamespacedName})
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ctrl.Result{}))

			Expect(s.Len()).To(Equal(0))
		},
		Entry("configmap is missing", func() {
			Expect(c.Delete(ctx, configmap)).To(Succeed())
		}),
		Entry("configmap is missing data key", func() {
			delete(configmap.Data, "ca.crt")
			Expect(c.Update(ctx, configmap)).To(Succeed())
		}),
		Entry("configmap has wrong 'discovery.gardener.cloud/public' label value", func() {
			configmap.Labels["discovery.gardener.cloud/public"] = "wrong"
			Expect(c.Update(ctx, configmap)).To(Succeed())
		}),
		Entry("configmap has wrong 'gardener.cloud/update-restriction' label value", func() {
			configmap.Labels["gardener.cloud/update-restriction"] = "false"
			Expect(c.Update(ctx, configmap)).To(Succeed())
		}),
		Entry("namespace is missing", func() {
			Expect(c.Delete(ctx, namespace)).To(Succeed())
		}),
		Entry("namespace is missing label", func() {
			delete(namespace.Labels, "project.gardener.cloud/name")
			Expect(c.Update(ctx, namespace)).To(Succeed())
		}),
		Entry("project is missing", func() {
			Expect(c.Delete(ctx, project)).To(Succeed())
		}),
		Entry("project is missing spec.namespace", func() {
			project.Spec.Namespace = nil
			Expect(c.Update(ctx, project)).To(Succeed())
		}),
		Entry("project spec.namespace does not match namespace name", func() {
			project.Spec.Namespace = ptr.To("wrong")
			Expect(c.Update(ctx, project)).To(Succeed())
		}),
		Entry("configmap is missing shoot name label", func() {
			delete(configmap.Labels, "shoot.gardener.cloud/name")
			Expect(c.Update(ctx, configmap)).To(Succeed())
		}),
		Entry("configmap is missing shoot uid label", func() {
			delete(configmap.Labels, "shoot.gardener.cloud/uid")
			Expect(c.Update(ctx, configmap)).To(Succeed())
		}),
		Entry("shoot is missing", func() {
			Expect(c.Delete(ctx, shoot)).To(Succeed())
		}),
		Entry("shoot uid does not match", func() {
			configmap.Labels["shoot.gardener.cloud/uid"] = "wrong"
			Expect(c.Update(ctx, configmap)).To(Succeed())
		}),
		Entry("pem block is of wrong type", func() {
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).ToNot(HaveOccurred())
			pemEncoded := &bytes.Buffer{}
			err = pem.Encode(pemEncoded, &pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
			})
			Expect(err).ToNot(HaveOccurred())
			configmap.Data["ca.crt"] = pemEncoded.String()
			Expect(c.Update(ctx, configmap)).To(Succeed())
		}),
	)
})
