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

package service

import (
	"fmt"
	"testing"

	operatorv1alpha1 "github.com/projectcontour/contour-operator/api/v1alpha1"
	objcontour "github.com/projectcontour/contour-operator/internal/objects/contour"
	objcfg "github.com/projectcontour/contour-operator/internal/objects/sharedconfig"

	contourv1alpha1 "github.com/projectcontour/contour/apis/projectcontour/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func checkServiceHasPort(t *testing.T, svc *corev1.Service, port int32) {
	t.Helper()

	for _, p := range svc.Spec.Ports {
		if p.Port == port {
			return
		}
	}
	t.Errorf("service is missing port %q", port)
}

func checkServiceHasTargetPort(t *testing.T, svc *corev1.Service, port int32) {
	t.Helper()

	intStrPort := intstr.IntOrString{IntVal: port}
	for _, p := range svc.Spec.Ports {
		if p.TargetPort == intStrPort {
			return
		}
	}
	t.Errorf("service is missing targetPort %d", port)
}

func checkServiceHasPortName(t *testing.T, svc *corev1.Service, name string) {
	t.Helper()

	for _, p := range svc.Spec.Ports {
		if p.Name == name {
			return
		}
	}
	t.Errorf("service is missing port name %q", name)
}

func checkServiceHasPortProtocol(t *testing.T, svc *corev1.Service, protocol corev1.Protocol) {
	t.Helper()

	for _, p := range svc.Spec.Ports {
		if p.Protocol == protocol {
			return
		}
	}
	t.Errorf("service is missing port protocol %q", protocol)
}

func checkServiceHasLabel(t *testing.T, svc *corev1.Service, key, val string) {
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

func checkServiceHasType(t *testing.T, svc *corev1.Service, svcType corev1.ServiceType) {
	t.Helper()

	if svc.Spec.Type != svcType {
		t.Errorf("service is missing type %s", svcType)
	}
}

func TestDesiredContourService(t *testing.T) {
	name := "svc-test"
	cfg := objcontour.Config{
		Name:        name,
		Namespace:   fmt.Sprintf("%s-ns", name),
		SpecNs:      "projectcontour",
		RemoveNs:    false,
		NetworkType: contourv1alpha1.LoadBalancerServicePublishingType,
	}
	cntr := objcontour.New(cfg)
	svc := DesiredContourService(cntr)
	xdsPort := objcfg.XDSPort
	checkServiceHasType(t, svc, corev1.ServiceTypeClusterIP)
	checkServiceHasLabel(t, svc, operatorv1alpha1.OwningContourNsLabel, cntr.Namespace)
	checkServiceHasLabel(t, svc, operatorv1alpha1.OwningContourNameLabel, cntr.Name)
	checkServiceHasPort(t, svc, xdsPort)
	checkServiceHasTargetPort(t, svc, xdsPort)
	checkServiceHasPortName(t, svc, "xds")
	checkServiceHasPortProtocol(t, svc, corev1.ProtocolTCP)
}
