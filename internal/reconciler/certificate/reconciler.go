// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package certificate

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"sync"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/controllerutils"
	secretsutils "github.com/gardener/gardener/pkg/utils/secrets"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/gardener-discovery-server/internal/store"
	"github.com/gardener/gardener-discovery-server/internal/store/certificate"
)

// Reconciler reconciles configmap objects that contain shoot CA.
type Reconciler struct {
	once         sync.Once
	storeMapping map[string]string
	mutex        sync.Mutex

	Client       client.Client
	ResyncPeriod time.Duration
	Store        store.Writer[certificate.Data]
}

// Reconcile retrieves the CA bundle info from a configmap and stores into cache.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var (
		log        = logf.FromContext(ctx)
		mappingKey = req.NamespacedName.String()
	)

	r.init()

	ctx, cancel := controllerutils.GetMainReconciliationContext(ctx, r.ResyncPeriod)
	defer cancel()

	configmap := &corev1.ConfigMap{}
	if err := r.Client.Get(ctx, req.NamespacedName, configmap); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Removing certificates from store - configmap not found")
			r.deleteMapping(mappingKey)

			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if configmap.DeletionTimestamp != nil {
		log.Info("Removing certificates from store - deletion timestamp present")
		r.deleteMapping(mappingKey)

		return reconcile.Result{}, nil
	}

	var (
		data string
		ok   bool
	)

	if data, ok = configmap.Data[secretsutils.DataKeyCertificateCA]; !ok || len(data) == 0 {
		log.Info("Removing certificates from store - configmap is missing data key", "key", secretsutils.DataKeyCertificateCA)
		r.deleteMapping(mappingKey)

		return reconcile.Result{}, nil
	}

	if configmap.Labels["discovery.gardener.cloud/public"] != "shoot-ca" ||
		configmap.Labels["gardener.cloud/update-restriction"] != "true" {
		log.Info("Removing certificates from store - configmap does not have expected labels or their value is incorrect")
		r.deleteMapping(mappingKey)

		return reconcile.Result{}, nil
	}

	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: req.Namespace}}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(namespace), namespace); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Removing certificates from store - namespace not found")
			r.deleteMapping(mappingKey)

			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	var (
		shootName   string
		shootUID    string
		projectName string
	)

	projectName, ok = namespace.Labels[v1beta1constants.ProjectName]
	if !ok {
		log.Info("Removing certificates from store - namespace does not have expected label", "label", v1beta1constants.ProjectName)
		r.deleteMapping(mappingKey)
		return reconcile.Result{}, nil
	}

	project := &gardencorev1beta1.Project{ObjectMeta: metav1.ObjectMeta{Name: projectName}}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(project), project); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Removing certificates from store - project not found", "project", projectName)
			r.deleteMapping(mappingKey)

			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if project.Spec.Namespace == nil {
		log.Info("Removing certificates from store - project spec.namespace is nil", "project", projectName)
		r.deleteMapping(mappingKey)

		return reconcile.Result{}, nil
	}

	if *project.Spec.Namespace != namespace.Name {
		log.Info("Removing certificates from store - namespace name does not match project namespace", "project", projectName)
		r.deleteMapping(mappingKey)

		return reconcile.Result{}, nil
	}

	shootName, ok = configmap.Labels[v1beta1constants.LabelShootName]
	if !ok {
		log.Info("Removing certificates from store - configmap does not have expected label", "label", v1beta1constants.LabelShootName)
		r.deleteMapping(mappingKey)
		return reconcile.Result{}, nil
	}

	shootUID, ok = configmap.Labels[v1beta1constants.ShootUID]
	if !ok {
		log.Info("Removing certificates from store - configmap does not have expected label", "label", v1beta1constants.ShootUID)
		r.deleteMapping(mappingKey)
		return reconcile.Result{}, nil
	}

	shoot := &gardencorev1beta1.Shoot{ObjectMeta: metav1.ObjectMeta{
		Name:      shootName,
		Namespace: req.Namespace,
	}}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(shoot), shoot); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Removing certificates from store - shoot not found", "shoot", shoot)
			r.deleteMapping(mappingKey)

			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if shootUID != string(shoot.UID) {
		log.Info("Removing certificates from store - shoot UID is different in spec and configmap label", "shoot", shoot)
		r.deleteMapping(mappingKey)
		return reconcile.Result{}, nil
	}

	certs := []byte(data)
	for {
		block, rest := pem.Decode(certs)
		if block == nil {
			break
		}

		if block.Type != "CERTIFICATE" {
			log.Info("Removing certificates from store - block type is not CERTIFICATE")
			r.deleteMapping(mappingKey)
			return reconcile.Result{}, nil
		}

		if len(block.Headers) > 0 {
			log.Info("Removing certificates from store - block headers are not expected")
			r.deleteMapping(mappingKey)
			return reconcile.Result{}, nil
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			log.Error(err, "Removing certificates from store - failed to parse certificate")
			r.deleteMapping(mappingKey)
			return reconcile.Result{}, nil
		}

		if !cert.IsCA {
			log.Info("Removing certificates from store - certificate is not a CA")
			r.deleteMapping(mappingKey)
			return reconcile.Result{}, nil
		}

		certs = rest
	}

	// Finally write the certificates to store
	log.Info("Adding certificates to store", "shoot", shoot)
	bundle := struct {
		Certs string `json:"certs"`
	}{Certs: data}

	payload, err := json.Marshal(bundle)
	if err != nil {
		r.deleteMapping(mappingKey)
		return ctrl.Result{}, err
	}

	r.createMapping(mappingKey, projectName+"--"+shootUID, payload)

	return ctrl.Result{RequeueAfter: r.ResyncPeriod}, nil
}

func (r *Reconciler) init() {
	r.once.Do(func() {
		r.storeMapping = make(map[string]string)
	})
}

func (r *Reconciler) createMapping(mapKey, dataKey string, data []byte) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.storeMapping[mapKey] = dataKey
	r.Store.Write(dataKey, certificate.Data{
		CABundle: data,
	})
}

func (r *Reconciler) deleteMapping(key string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.Store.Delete(r.storeMapping[key])
	delete(r.storeMapping, key)
}
