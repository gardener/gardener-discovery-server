// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openidmeta

import (
	"context"
	"time"

	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler reconciles secret objects that contain
// shoot openid configuration and JWKS.
type Reconciler struct {
	Client       client.Client
	Log          logr.Logger
	ResyncPeriod time.Duration
}

// Reconcile retrieves the public OIDC metadata info from a secret and stores into cache.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	ctx, cancel := controllerutils.GetMainReconciliationContext(ctx, r.ResyncPeriod)
	defer cancel()

	log.Info("Reconciling")
	defer log.Info("Reconcile finished")

	secret := &corev1.Secret{}
	if err := r.Client.Get(ctx, req.NamespacedName, secret); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Not found - removing metadata from store")
			// TODO delete from cache

			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if secret.DeletionTimestamp != nil {
		log.Info("Deletion timestamp present - removing metadata from store")

		// TODO delete from cache

		return reconcile.Result{}, nil
	}

	// TODO
	// run the necessary checks to ensure that there is a shoot that has enabled this feature
	// check that the keys in the secret are indeed public

	return ctrl.Result{RequeueAfter: r.ResyncPeriod}, nil
}
