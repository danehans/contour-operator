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
	"context"
	"fmt"

	operatorv1alpha1 "github.com/projectcontour/contour-operator/api/v1alpha1"
	"github.com/projectcontour/contour-operator/internal/equality"
	objcontour "github.com/projectcontour/contour-operator/internal/objects/contour"
	"github.com/projectcontour/contour-operator/pkg/labels"

	contourv1alpha1 "github.com/projectcontour/contour/apis/projectcontour/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Ensure ensures that an Envoy resource exists for the provided contour.
func Ensure(ctx context.Context, cli client.Client, contour *operatorv1alpha1.Contour) error {
	desired := Desired(contour)
	current, err := current(ctx, cli, contour)
	if err != nil {
		if errors.IsNotFound(err) {
			return create(ctx, cli, desired)
		}
		return fmt.Errorf("failed to get envoy %s/%s: %w", desired.Namespace, desired.Name, err)
	}
	if err := updateIfNeeded(ctx, cli, contour, current, desired); err != nil {
		return fmt.Errorf("failed to update envoy %s/%s: %w", desired.Namespace, desired.Name, err)
	}
	return nil
}

// EnsureDeleted ensures that an Envoy resource for the provided contour
// is deleted if Contour owner labels exist.
func EnsureDeleted(ctx context.Context, cli client.Client, contour *operatorv1alpha1.Contour) error {
	svc, err := current(ctx, cli, contour)
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

// Desired generates the desired Envoy resource for the provided contour.
func Desired(contour *operatorv1alpha1.Contour) *contourv1alpha1.Envoy {
	return &contourv1alpha1.Envoy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   contour.Spec.Namespace.Name,
			Name:        contour.Name,
			Annotations: map[string]string{},
			Labels: map[string]string{
				operatorv1alpha1.OwningContourNameLabel: contour.Name,
				operatorv1alpha1.OwningContourNsLabel:   contour.Namespace,
			},
		},
		Spec: contourv1alpha1.EnvoySpec{
			NetworkPublishing: contourv1alpha1.NetworkPublishing{
				Type:           contour.Spec.NetworkPublishing.Envoy.Type,
				LoadBalancer:   contour.Spec.NetworkPublishing.Envoy.LoadBalancer,
				ContainerPorts: contour.Spec.NetworkPublishing.Envoy.ContainerPorts,
			},
		},
	}
}

// current returns the current Envoy resource for the provided contour.
func current(ctx context.Context, cli client.Client, contour *operatorv1alpha1.Contour) (*contourv1alpha1.Envoy, error) {
	current := &contourv1alpha1.Envoy{}
	key := types.NamespacedName{
		Namespace: contour.Spec.Namespace.Name,
		Name:      contour.Name,
	}
	err := cli.Get(ctx, key, current)
	if err != nil {
		return nil, err
	}
	return current, nil
}

// create creates an Envoy resource for the provided envoy.
func create(ctx context.Context, cli client.Client, envoy *contourv1alpha1.Envoy) error {
	if err := cli.Create(ctx, envoy); err != nil {
		return fmt.Errorf("failed to create envoy %s/%s: %w", envoy.Namespace, envoy.Name, err)
	}
	return nil
}

// updateIfNeeded updates an Envoy resource if current doesn't match desired, using
// contour to ensure ownership labels exist.
func updateIfNeeded(ctx context.Context, cli client.Client, contour *operatorv1alpha1.Contour, current, desired *contourv1alpha1.Envoy) error {
	if labels.Exist(current, objcontour.OwnerLabels(contour)) {
		updated := equality.EnvoyChanged(current, desired)
		if updated {
			if err := cli.Update(ctx, desired); err != nil {
				return fmt.Errorf("failed to update envoy %s/%s: %w", desired.Namespace, desired.Name, err)
			}
			return nil
		}
	}
	return nil
}
