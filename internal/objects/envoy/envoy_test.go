// Copyright Project Contour Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package envoy

import (
	"testing"

	operatorv1alpha1 "github.com/projectcontour/contour-operator/api/v1alpha1"

	contourv1alpha1 "github.com/projectcontour/contour/apis/projectcontour/v1alpha1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
)

func checkEnvoyHasLabel(t *testing.T, svc *contourv1alpha1.Envoy, key, val string) {
	t.Helper()

	if svc.Labels == nil {
		t.Errorf("service is missing labels")
	}
	v, ok := svc.Labels[key]
	if ok {
		if v == val {
			return
		}
	}

	t.Errorf("service is missing label %s=%s", key, val)
}

func checkEnvoyHasNetPub(t *testing.T, envoy *contourv1alpha1.Envoy, pub contourv1alpha1.NetworkPublishing) {
	t.Helper()

	if !apiequality.Semantic.DeepEqual(envoy.Spec.NetworkPublishing, pub) {
		t.Error("envoy has incorrect network publishing")
	}
}

func TestDesired(t *testing.T) {
	cntr := &operatorv1alpha1.Contour{
		Spec: operatorv1alpha1.ContourSpec{
			NetworkPublishing: operatorv1alpha1.NetworkPublishing{
				Envoy: contourv1alpha1.NetworkPublishing{
					Type: contourv1alpha1.LoadBalancerServicePublishingType,
					LoadBalancer: contourv1alpha1.LoadBalancerStrategy{
						Scope: contourv1alpha1.ExternalLoadBalancer,
						ProviderParameters: contourv1alpha1.ProviderLoadBalancerParameters{
							Type: contourv1alpha1.AWSLoadBalancerProvider,
						},
					},
					ContainerPorts: []contourv1alpha1.ContainerPort{
						{
							Name:       "http",
							PortNumber: int32(8080),
						},
						{
							Name:       "https",
							PortNumber: int32(8443),
						},
					},
				},
			},
		},
	}
	envoy := Desired(cntr)
	checkEnvoyHasLabel(t, envoy, operatorv1alpha1.OwningContourNsLabel, cntr.Namespace)
	checkEnvoyHasLabel(t, envoy, operatorv1alpha1.OwningContourNameLabel, cntr.Name)
	checkEnvoyHasNetPub(t, envoy, cntr.Spec.NetworkPublishing.Envoy)
}
