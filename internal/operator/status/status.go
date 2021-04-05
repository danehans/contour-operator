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

package status

import (
	"context"
	"fmt"

	operatorv1alpha1 "github.com/projectcontour/contour-operator/api/v1alpha1"
	"github.com/projectcontour/contour-operator/internal/equality"
	objds "github.com/projectcontour/contour-operator/internal/objects/daemonset"
	objdeploy "github.com/projectcontour/contour-operator/internal/objects/deployment"
	retryable "github.com/projectcontour/contour-operator/internal/retryableerror"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// syncContourStatus computes the current status of contour and updates status upon
// any changes since last sync.
func SyncContour(ctx context.Context, cli client.Client, contour *operatorv1alpha1.Contour) error {
	var err error
	var errs []error

	updated := contour.DeepCopy()
	deploy, err := objdeploy.CurrentDeployment(ctx, cli, updated)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to get deployment for contour %s/%s status: %w",
			updated.Namespace, updated.Name, err))
	} else {
		updated.Status.AvailableContours = deploy.Status.AvailableReplicas
	}
	ds, err := objds.CurrentDaemonSet(ctx, cli, updated)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to get daemonset for contour %s/%s status: %w",
			updated.Namespace, updated.Name, err))
	} else {
		updated.Status.AvailableEnvoys = ds.Status.NumberAvailable
	}

	updated.Status.Conditions = mergeConditions(updated.Status.Conditions, computeContourAvailableCondition(deploy, ds))

	if equality.ContourStatusChanged(contour.Status, updated.Status) {
		if err := cli.Status().Update(ctx, updated); err != nil {
			errs = append(errs, fmt.Errorf("failed to update contour %s/%s status: %w",
				updated.Namespace, updated.Name, err))
		}
	}

	return retryable.NewMaybeRetryableAggregate(errs)
}
