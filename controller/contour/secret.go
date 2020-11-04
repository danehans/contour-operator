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

package contour

import (
	"context"
	"fmt"

	operatorv1alpha1 "github.com/projectcontour/contour-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// tlsSecretsAsNsNames returns a map of TLS secrets for the provided contour.
func (r *Reconciler) tlsSecretsAsNsNames(ctx context.Context, contour *operatorv1alpha1.Contour) ([]string, error) {
	current := &corev1.SecretList{}
	if err := r.Client.List(ctx, current, client.InNamespace(contour.Spec.Namespace.Name)); err != nil {
		return nil, fmt.Errorf("failed to list current for contour %s/%s", contour.Namespace, contour.Name)
	}

	var secrets []string
	for _, s := range current.Items {
		if s.Type == corev1.SecretTypeTLS {
			secrets = append(secrets, s.Name)
		}
	}
	return secrets, nil
}
