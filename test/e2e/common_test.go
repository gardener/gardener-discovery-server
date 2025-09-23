// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"context"
	"net/http"
	"os"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/test/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

const projectNamespace = "garden-local"

func defaultShootCreationFramework() *framework.ShootCreationFramework {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	return framework.NewShootCreationFramework(&framework.ShootCreationConfig{
		GardenerConfig: &framework.GardenerConfig{
			ProjectNamespace:   projectNamespace,
			GardenerKubeconfig: kubeconfigPath,
			SkipAccessingShoot: true,
			CommonConfig:       &framework.CommonConfig{},
		},
	})
}

func defaultShoot(generateName string) *gardencorev1beta1.Shoot {
	return &gardencorev1beta1.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: generateName,
			Annotations: map[string]string{
				v1beta1constants.AnnotationShootCloudConfigExecutionMaxDelaySeconds: "0",
			},
		},
		Spec: gardencorev1beta1.ShootSpec{
			Region:            "local",
			SecretBindingName: ptr.To("local"),
			CloudProfile:      &gardencorev1beta1.CloudProfileReference{Name: "local"},
			Kubernetes: gardencorev1beta1.Kubernetes{
				Version:       "1.33.0",
				KubeAPIServer: &gardencorev1beta1.KubeAPIServerConfig{},
			},
			Networking: &gardencorev1beta1.Networking{
				Type:           ptr.To("calico"),
				Nodes:          ptr.To("10.0.0.0/16"),
				ProviderConfig: &runtime.RawExtension{Raw: []byte(`{"apiVersion":"calico.networking.extensions.gardener.cloud/v1alpha1","kind":"NetworkConfig","typha":{"enabled":false},"backend":"none"}`)},
			},
			Provider: gardencorev1beta1.Provider{
				Type: "local",
				Workers: []gardencorev1beta1.Worker{{
					Name: "local",
					Machine: gardencorev1beta1.Machine{
						Type: "local",
					},
					CRI: &gardencorev1beta1.CRI{
						Name: gardencorev1beta1.CRINameContainerD,
					},
					Minimum: 1,
					Maximum: 1,
				}},
			},
		},
	}
}

func addAnnotations(shoot *gardencorev1beta1.Shoot) error {
	shoot.Annotations[v1beta1constants.AnnotationAuthenticationIssuer] = v1beta1constants.AnnotationAuthenticationIssuerManaged
	shoot.Annotations[v1beta1constants.GardenerOperation] = v1beta1constants.GardenerOperationReconcile
	return nil
}

func getJWKSForShoot(ctx context.Context, shootUID types.UID) (*http.Response, error) {
	uri := discoveryServerBaseURI + "/projects/local/shoots/" + string(shootUID) + "/issuer/jwks"
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	return discoveryClient.Do(req)
}

func getWellKnownForShoot(ctx context.Context, shootUID types.UID) (*http.Response, error) {
	uri := discoveryServerBaseURI + "/projects/local/shoots/" + string(shootUID) + "/issuer/.well-known/openid-configuration"
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	return discoveryClient.Do(req)
}

func getCABundleForShoot(ctx context.Context, shootUID types.UID) (*http.Response, error) {
	uri := discoveryServerBaseURI + "/projects/local/shoots/" + string(shootUID) + "/cluster-ca"
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	return discoveryClient.Do(req)
}
