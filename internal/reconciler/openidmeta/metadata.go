// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openidmeta

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gardener/gardener-discovery-server/internal/store/openidmeta"
	"github.com/gardener/gardener-discovery-server/internal/utils"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/go-jose/go-jose/v4"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler reconciles secret objects that contain
// shoot openid configuration and JWKS.
type Reconciler struct {
	Client       client.Client
	ResyncPeriod time.Duration
	Store        openidmeta.Writer
}

// Reconcile retrieves the public OIDC metadata info from a secret and stores into cache.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	ctx, cancel := controllerutils.GetMainReconciliationContext(ctx, r.ResyncPeriod)
	defer cancel()

	secret := &corev1.Secret{}
	if err := r.Client.Get(ctx, req.NamespacedName, secret); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Removing metadata from store - secret not found")
			r.Store.Delete(req.Name)

			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if secret.DeletionTimestamp != nil {
		log.Info("Removing metadata from store - deletion timestamp present")
		r.Store.Delete(req.Name)

		return reconcile.Result{}, nil
	}

	const (
		openidConfigKey = "openid-config"
		jwksKey         = "jwks"
	)
	if v, ok := secret.Data[openidConfigKey]; !ok || len(v) == 0 {
		log.Info("Removing metadata from store - secret is missing data key", "key", openidConfigKey)
		r.Store.Delete(req.Name)

		return reconcile.Result{}, nil
	}

	if v, ok := secret.Data[jwksKey]; !ok || len(v) == 0 {
		log.Info("Removing metadata from store - secret is missing data key", "key", jwksKey)
		r.Store.Delete(req.Name)

		return reconcile.Result{}, nil
	}

	labels := secret.GetLabels()
	if v, ok := labels[v1beta1constants.LabelPublicKeys]; !ok || v != v1beta1constants.LabelPublicKeysServiceAccount {
		log.Info("Removing metadata from store - secret does not have expected label", "label", v1beta1constants.LabelPublicKeys)
		r.Store.Delete(req.Name)
		return reconcile.Result{}, nil
	}

	var (
		shootName      string
		shootNamespace string
		projectName    string
		ok             bool
	)

	projectName, ok = labels[v1beta1constants.ProjectName]
	if !ok {
		log.Info("Removing metadata from store - secret does not have expected label", "label", v1beta1constants.ProjectName)
		r.Store.Delete(req.Name)
		return reconcile.Result{}, nil
	}

	shootName, ok = labels[v1beta1constants.LabelShootName]
	if !ok {
		log.Info("Removing metadata from store - secret does not have expected label", "label", v1beta1constants.LabelShootName)
		r.Store.Delete(req.Name)
		return reconcile.Result{}, nil
	}

	shootNamespace, ok = labels[v1beta1constants.LabelShootNamespace]
	if !ok {
		log.Info("Removing metadata from store - secret does not have expected label", "label", v1beta1constants.LabelShootNamespace)
		r.Store.Delete(req.Name)
		return reconcile.Result{}, nil
	}

	projName, shootUID, err := utils.SplitProjectNameAndShootUID(req.Name)
	if err != nil {
		log.Error(err, "Removing metadata from store - secret name is not in the correct format")
		r.Store.Delete(req.Name)
		return reconcile.Result{}, nil
	}

	if projectName != projName {
		log.Info("Removing metadata from store - project name does not match between secret name and the project label")
		r.Store.Delete(req.Name)
		return reconcile.Result{}, nil
	}

	project := &gardencorev1beta1.Project{ObjectMeta: metav1.ObjectMeta{Name: projectName}}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(project), project); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Removing metadata from store - project not found", "project", projectName)
			r.Store.Delete(req.Name)

			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if project.Spec.Namespace == nil {
		log.Info("Removing metadata from store - project spec.namespace is nil", "project", projectName)
		r.Store.Delete(req.Name)

		return reconcile.Result{}, nil
	}

	if shootNamespace != *project.Spec.Namespace {
		log.Info("Removing metadata from store - secret shoot namespace label does not match project namespace", "project", projectName)
		r.Store.Delete(req.Name)

		return reconcile.Result{}, nil
	}

	shoot := &gardencorev1beta1.Shoot{ObjectMeta: metav1.ObjectMeta{
		Name:      shootName,
		Namespace: shootNamespace,
	}}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(shoot), shoot); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Removing metadata from store - shoot not found", "shoot", client.ObjectKeyFromObject(shoot))
			r.Store.Delete(req.Name)

			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if shootUID != string(shoot.UID) {
		log.Info("Removing metadata from store - shoot UID is different in spec and in secret name", "shoot", client.ObjectKeyFromObject(shoot))
		r.Store.Delete(req.Name)

		return reconcile.Result{}, nil
	}

	if v, ok := shoot.Annotations[v1beta1constants.AnnotationAuthenticationIssuer]; !ok || v != v1beta1constants.AnnotationAuthenticationIssuerManaged {
		log.Info("Removing metadata from store - shoot managed issuer annotation is missing", "shoot", client.ObjectKeyFromObject(shoot))
		r.Store.Delete(req.Name)

		return reconcile.Result{}, nil
	}

	// a best effort check to ensure that URIs use https
	var cfg config
	err = json.Unmarshal(secret.Data[openidConfigKey], &cfg)
	if err != nil {
		log.Error(err, "Removing metadata from store - cannot unmarshal openid-config")
		r.Store.Delete(req.Name)

		return reconcile.Result{}, nil
	}

	if !strings.HasPrefix(cfg.Issuer, "https://") || !strings.HasPrefix(cfg.JWKSURI, "https://") {
		log.Error(err, "Removing metadata from store - open ID config is invalid, either issuer or jwks_uri does not start with https://")
		r.Store.Delete(req.Name)

		return reconcile.Result{}, nil
	}

	keySet, err := loadKeySet(secret.Data[jwksKey])
	if err != nil {
		log.Error(err, "Removing metadata from store - failed parsing JWKS")
		r.Store.Delete(req.Name)

		return reconcile.Result{}, nil
	}

	// a check if for some reason there is a non public key in there
	for _, k := range keySet.Keys {
		if !k.IsPublic() {
			log.Info("Removing metadata from store - found a non public key in JWKS")
			r.Store.Delete(req.Name)

			return reconcile.Result{}, nil
		}

		if !k.Valid() {
			log.Info("Removing metadata from store - found an invalid key in JWKS")
			r.Store.Delete(req.Name)

			return reconcile.Result{}, nil
		}
	}

	// Finally write the metadata to store
	log.Info("Adding metadata to store")
	r.Store.Write(req.Name, openidmeta.Data{
		Config: secret.Data[openidConfigKey],
		JWKS:   secret.Data[jwksKey],
	})

	return ctrl.Result{RequeueAfter: r.ResyncPeriod}, nil
}

type config struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

// loadKeySet parses the jwks key set.
func loadKeySet(jwks []byte) (*jose.JSONWebKeySet, error) {
	var keySet jose.JSONWebKeySet
	err := json.Unmarshal(jwks, &keySet)
	if err != nil {
		return nil, err
	}

	return &keySet, nil
}
