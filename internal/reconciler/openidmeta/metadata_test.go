// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openidmeta_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"slices"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/go-jose/go-jose/v4"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	oidreconciler "github.com/gardener/gardener-discovery-server/internal/reconciler/openidmeta"
	oidstore "github.com/gardener/gardener-discovery-server/internal/store/openidmeta"
)

var _ = Describe("#ReconcileOpenIDMeta", func() {
	var (
		reconciler *oidreconciler.Reconciler

		c     client.Client
		store *oidstore.Store

		shoot                *gardencorev1beta1.Shoot
		project              *gardencorev1beta1.Project
		secret               *corev1.Secret
		secretNamespacedName types.NamespacedName

		ctx            = context.TODO()
		shootName      = "my-shoot"
		shootNamespace = "garden-abc"
		shootUID       = types.UID("7a25a9b8-f7fc-4e1e-a421-31b4deaa3086")
		resyncPeriod   = time.Second

		expectStoreEntry = func(store *oidstore.Store, key string, want oidstore.Data) {
			got, ok := store.Read(key)
			Expect(ok).To(BeTrue())
			Expect(got).To(Equal(want))
		}

		generateKeySet = func() (*jose.JSONWebKeySet, error) {
			key, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				return nil, err
			}

			privateKey := jose.JSONWebKey{Key: key, KeyID: "", Algorithm: string(jose.RS256), Use: "sig"}
			thumb, err := privateKey.Thumbprint(crypto.SHA256)
			if err != nil {
				return nil, err
			}
			kid := base64.URLEncoding.EncodeToString(thumb)
			privateKey.KeyID = kid

			keySet := &jose.JSONWebKeySet{}
			publicKey := jose.JSONWebKey{Key: key.Public(), KeyID: privateKey.KeyID, Algorithm: string(jose.RS256), Use: "sig"}
			keySet.Keys = append(keySet.Keys, publicKey)
			return keySet, nil
		}

		keySet            *jose.JSONWebKeySet
		expectedJWKSBytes []byte
	)

	BeforeEach(func() {
		c = fake.NewClientBuilder().WithScheme(kubernetes.GardenScheme).Build()
		shoot = &gardencorev1beta1.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      shootName,
				Namespace: shootNamespace,
				UID:       shootUID,
				Annotations: map[string]string{
					"authentication.gardener.cloud/issuer": "managed",
				},
			},
		}
		project = &gardencorev1beta1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "abc",
			},
			Spec: gardencorev1beta1.ProjectSpec{
				Namespace: ptr.To(shootNamespace),
			},
		}

		var err error
		keySet, err = generateKeySet()
		Expect(err).ToNot(HaveOccurred())

		jwksBytes, err := json.Marshal(keySet)
		Expect(err).ToNot(HaveOccurred())
		expectedJWKSBytes = slices.Clone(jwksBytes)

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      project.Name + "--" + string(shoot.UID),
				Namespace: "gardener-system-shoot-issuer",
				Labels: map[string]string{
					"authentication.gardener.cloud/public-keys": "serviceaccount",
					"project.gardener.cloud/name":               project.Name,
					"shoot.gardener.cloud/name":                 shoot.Name,
					"shoot.gardener.cloud/namespace":            shoot.Namespace,
				},
			},
			Data: map[string][]byte{
				"openid-config": []byte(`{"issuer":"https://foo","jwks_uri":"https://foo/jwks"}`),
				"jwks":          jwksBytes,
			},
		}
		store = oidstore.NewStore()
		reconciler = &oidreconciler.Reconciler{
			Client:       c,
			Store:        store,
			ResyncPeriod: resyncPeriod,
		}
		secretNamespacedName = client.ObjectKeyFromObject(secret)
	})

	It("should write entry to store", func() {
		Expect(c.Create(ctx, project)).To(Succeed())
		Expect(c.Create(ctx, shoot)).To(Succeed())
		Expect(c.Create(ctx, secret)).To(Succeed())

		res, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(secret)})
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(ctrl.Result{RequeueAfter: resyncPeriod}))

		Expect(store.Len()).To(Equal(1))
		expectStoreEntry(store, secret.Name, oidstore.Data{
			Config: []byte(`{"issuer":"https://foo","jwks_uri":"https://foo/jwks"}`),
			JWKS:   expectedJWKSBytes,
		})
	})

	DescribeTable(
		"should remove entry from store because of failed validation",
		func(prepFunc func()) {
			Expect(c.Create(ctx, project)).To(Succeed())
			Expect(c.Create(ctx, shoot)).To(Succeed())
			Expect(c.Create(ctx, secret)).To(Succeed())

			res, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: secretNamespacedName})
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ctrl.Result{RequeueAfter: resyncPeriod}))

			Expect(store.Len()).To(Equal(1))
			expectStoreEntry(store, secret.Name, oidstore.Data{
				Config: []byte(`{"issuer":"https://foo","jwks_uri":"https://foo/jwks"}`),
				JWKS:   expectedJWKSBytes,
			})

			prepFunc()

			res, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: secretNamespacedName})
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ctrl.Result{}))

			Expect(store.Len()).To(Equal(0))
		},
		Entry("secret is missing", func() {
			Expect(c.Delete(ctx, secret)).To(Succeed())
		}),
		Entry("secret is not labeled with authentication.gardener.cloud/public-keys", func() {
			delete(secret.Labels, "authentication.gardener.cloud/public-keys")
			Expect(c.Update(ctx, secret)).To(Succeed())
		}),
		Entry("secret label authentication.gardener.cloud/public-keys does not have correct value", func() {
			secret.Labels["authentication.gardener.cloud/public-keys"] = "wrong"
			Expect(c.Update(ctx, secret)).To(Succeed())
		}),
		Entry("secret does not have openid-config data entry", func() {
			delete(secret.Data, "openid-config")
			Expect(c.Update(ctx, secret)).To(Succeed())
		}),
		Entry("secret does not have jwks data entry", func() {
			delete(secret.Data, "jwks")
			Expect(c.Update(ctx, secret)).To(Succeed())
		}),
		Entry("secret is not labeled with project.gardener.cloud/name", func() {
			delete(secret.Labels, "project.gardener.cloud/name")
			Expect(c.Update(ctx, secret)).To(Succeed())
		}),
		Entry("secret is not labeled with shoot.gardener.cloud/name", func() {
			delete(secret.Labels, "shoot.gardener.cloud/name")
			Expect(c.Update(ctx, secret)).To(Succeed())
		}),
		Entry("secret is not labeled with shoot.gardener.cloud/namespace", func() {
			delete(secret.Labels, "shoot.gardener.cloud/namespace")
			Expect(c.Update(ctx, secret)).To(Succeed())
		}),
		Entry("secret is labeled with wrong project name", func() {
			secret.Labels["project.gardener.cloud/name"] = "wrong"
			Expect(c.Update(ctx, secret)).To(Succeed())
		}),
		Entry("project is not found", func() {
			Expect(c.Delete(ctx, project)).To(Succeed())
		}),
		Entry("project namespace is missing", func() {
			newProject := project.DeepCopy()
			newProject.ResourceVersion = ""
			newProject.Spec.Namespace = nil
			Expect(c.Delete(ctx, project)).To(Succeed())
			Expect(c.Create(ctx, newProject)).To(Succeed())
		}),
		Entry("secret shoot namespace label is different from project namespace", func() {
			secret.Labels["shoot.gardener.cloud/namespace"] = "wrong"
			Expect(c.Update(ctx, secret)).To(Succeed())
		}),
		Entry("shoot is missing", func() {
			Expect(c.Delete(ctx, shoot))
		}),
		Entry("shoot has different UID", func() {
			newShoot := shoot.DeepCopy()
			newShoot.ResourceVersion = ""
			newShoot.UID = types.UID(uuid.New().String())
			Expect(c.Delete(ctx, shoot)).To(Succeed())
			Expect(c.Create(ctx, newShoot)).To(Succeed())
		}),
		Entry("shoot does not have the correct annotation", func() {
			shoot.Annotations["authentication.gardener.cloud/issuer"] = "wrong"
			Expect(c.Update(ctx, shoot)).To(Succeed())
		}),
		Entry("issuer in openid config does not start with https://", func() {
			secret.Data["openid-config"] = []byte(`{"issuer":"http://foo","jwks_uri":"https://foo/jwks"}`)
			Expect(c.Update(ctx, secret)).To(Succeed())
		}),
		Entry("jwks_uri in openid config does not start with https://", func() {
			secret.Data["openid-config"] = []byte(`{"issuer":"https://foo","jwks_uri":"http://foo/jwks"}`)
			Expect(c.Update(ctx, secret)).To(Succeed())
		}),
		Entry("jwks contains a private key", func() {
			key, err := rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).ToNot(HaveOccurred())

			privateKey := jose.JSONWebKey{Key: key, KeyID: "", Algorithm: string(jose.RS256), Use: "sig"}
			thumb, err := privateKey.Thumbprint(crypto.SHA256)
			Expect(err).ToNot(HaveOccurred())
			kid := base64.URLEncoding.EncodeToString(thumb)
			privateKey.KeyID = kid

			keySet := &jose.JSONWebKeySet{}
			err = json.Unmarshal(secret.Data["jwks"], keySet)
			Expect(err).ToNot(HaveOccurred())
			keySet.Keys = append(keySet.Keys, privateKey)

			jwksBytes, err := json.Marshal(keySet)
			Expect(err).ToNot(HaveOccurred())

			secret.Data["jwks"] = jwksBytes
			Expect(c.Update(ctx, secret)).To(Succeed())
		}),
		Entry("jwks contains an invalid key", func() {
			keySet := &jose.JSONWebKeySet{}
			err := json.Unmarshal(secret.Data["jwks"], keySet)
			Expect(err).ToNot(HaveOccurred())

			// this key will not be unmarshaled successfully
			pubKey := &rsa.PublicKey{
				N: &big.Int{},
				E: 0,
			}
			keySet.Keys[0].Key = pubKey

			jwksBytes, err := json.Marshal(keySet)
			Expect(err).ToNot(HaveOccurred())

			secret.Data["jwks"] = jwksBytes
			Expect(c.Update(ctx, secret)).To(Succeed())
		}),
	)
})
