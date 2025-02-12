// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package certificate

import (
	"time"

	"golang.org/x/time/rate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ControllerName is the name of the shoot CA controller.
const ControllerName = "shoot-ca"

// SetupWithManager specifies how the controller is built
// to watch configmaps that contain shoot CA bundle.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Client == nil {
		r.Client = mgr.GetClient()
	}

	return builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		For(&corev1.ConfigMap{}, builder.WithPredicates(configmapPredicate())).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 50,
			RateLimiter: workqueue.NewTypedMaxOfRateLimiter(
				workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](5*time.Second, 2*time.Minute),
				&workqueue.TypedBucketRateLimiter[reconcile.Request]{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
			),
		}).
		Complete(r)
}

func configmapPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return isRelevantConfigMap(e.Object) },
		UpdateFunc:  func(e event.UpdateEvent) bool { return isRelevantConfigMapUpdate(e.ObjectOld, e.ObjectNew) },
		DeleteFunc:  func(e event.DeleteEvent) bool { return isRelevantConfigMap(e.Object) },
		GenericFunc: func(_ event.GenericEvent) bool { return false },
	}
}

func isRelevantConfigMap(obj client.Object) bool {
	configmap, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return false
	}
	// we only allow update-restricted resources
	// it is not safe to read data from resources that might be modified by users
	// TODO: Use v1beta1constants.(LabelDiscoveryPublic|LabelUpdateRestriction) for label keys when gardener is updated to >= v1.112.0
	return configmap.Labels != nil &&
		configmap.Labels["discovery.gardener.cloud/public"] == "shoot-ca" &&
		configmap.Labels["gardener.cloud/update-restriction"] == "true"
}

func isRelevantConfigMapUpdate(oldObj, newObj client.Object) bool {
	return isRelevantConfigMap(newObj) || isRelevantConfigMap(oldObj)
}
