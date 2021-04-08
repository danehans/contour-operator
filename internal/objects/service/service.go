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
	"context"
	"fmt"

	operatorv1alpha1 "github.com/projectcontour/contour-operator/api/v1alpha1"
	"github.com/projectcontour/contour-operator/internal/equality"
	objcontour "github.com/projectcontour/contour-operator/internal/objects/contour"
	objdeploy "github.com/projectcontour/contour-operator/internal/objects/deployment"
	objcfg "github.com/projectcontour/contour-operator/internal/objects/sharedconfig"
	"github.com/projectcontour/contour-operator/pkg/labels"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// contourSvcName is the name of Contour's Service.
	// [TODO] danehans: Update Contour name to contour.Name + "-contour" to support multiple
	// Contours/ns when https://github.com/projectcontour/contour/issues/2122 is fixed.
	contourSvcName = "contour"
)

// EnsureContourService ensures that a Contour Service exists for the given contour.
func EnsureContourService(ctx context.Context, cli client.Client, contour *operatorv1alpha1.Contour) error {
	desired := DesiredContourService(contour)
	current, err := currentContourService(ctx, cli, contour)
	if err != nil {
		if errors.IsNotFound(err) {
			return createService(ctx, cli, desired)
		}
		return fmt.Errorf("failed to get service %s/%s: %w", desired.Namespace, desired.Name, err)
	}
	if err := updateContourServiceIfNeeded(ctx, cli, contour, current, desired); err != nil {
		return fmt.Errorf("failed to update service %s/%s: %w", desired.Namespace, desired.Name, err)
	}
	return nil
}

// EnsureContourServiceDeleted ensures that a Contour Service for the
// provided contour is deleted if Contour owner labels exist.
func EnsureContourServiceDeleted(ctx context.Context, cli client.Client, contour *operatorv1alpha1.Contour) error {
	svc, err := currentContourService(ctx, cli, contour)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if labels.Exist(svc, objcontour.OwnerLabels(contour)) {
		if err := cli.Delete(ctx, svc); err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
	}
	return nil
}

// DesiredContourService generates the desired Contour Service for the given contour.
func DesiredContourService(contour *operatorv1alpha1.Contour) *corev1.Service {
	xdsPort := objcfg.XDSPort
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: contour.Spec.Namespace.Name,
			Name:      contourSvcName,
			Labels: map[string]string{
				operatorv1alpha1.OwningContourNameLabel: contour.Name,
				operatorv1alpha1.OwningContourNsLabel:   contour.Namespace,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "xds",
					Port:       xdsPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.IntOrString{IntVal: xdsPort},
				},
			},
			Selector:        objdeploy.ContourDeploymentPodSelector().MatchLabels,
			Type:            corev1.ServiceTypeClusterIP,
			SessionAffinity: corev1.ServiceAffinityNone,
		},
	}
	return svc
}

// currentContourService returns the current Contour Service for the provided contour.
func currentContourService(ctx context.Context, cli client.Client, contour *operatorv1alpha1.Contour) (*corev1.Service, error) {
	current := &corev1.Service{}
	key := types.NamespacedName{
		Namespace: contour.Spec.Namespace.Name,
		Name:      contourSvcName,
	}
	err := cli.Get(ctx, key, current)
	if err != nil {
		return nil, err
	}
	return current, nil
}

// createService creates a Service resource for the provided svc.
func createService(ctx context.Context, cli client.Client, svc *corev1.Service) error {
	if err := cli.Create(ctx, svc); err != nil {
		return fmt.Errorf("failed to create service %s/%s: %w", svc.Namespace, svc.Name, err)
	}
	return nil
}

// updateContourServiceIfNeeded updates a Contour Service if current does not match desired.
func updateContourServiceIfNeeded(ctx context.Context, cli client.Client, contour *operatorv1alpha1.Contour, current, desired *corev1.Service) error {
	if labels.Exist(current, objcontour.OwnerLabels(contour)) {
		_, updated := equality.ClusterIPServiceChanged(current, desired)
		if updated {
			if err := cli.Update(ctx, desired); err != nil {
				return fmt.Errorf("failed to update service %s/%s: %w", desired.Namespace, desired.Name, err)
			}
			return nil
		}
	}
	return nil
}
