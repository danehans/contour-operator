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

package deployment

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"

	operatorv1alpha1 "github.com/projectcontour/contour-operator/api/v1alpha1"
	retryable "github.com/projectcontour/contour-operator/util/retryableerror"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var pointerTo = func(ios intstr.IntOrString) *intstr.IntOrString { return &ios }

// Config holds all the things necessary for the controller to run.
type Config struct {
	// ContourImage is the name of the Contour container image.
	ContourImage string
	// EnvoyImage is the name of the Envoy container image.
	EnvoyImage string
}

// Reconciler reconciles a Deployment object.
type Reconciler struct {
	Config Config
	Client client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=operator.projectcontour.io,resources=contours,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=operator.projectcontour.io,resources=contours/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;delete;create;update

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("deployment", req.NamespacedName)

	r.Log.Info("reconciling", "request", req)

	// Only proceed if we can get the state of deployment.
	deploy := &appsv1.Deployment{}
	if err := r.Client.Get(ctx, req.NamespacedName, deploy); err != nil {
		if errors.IsNotFound(err) {
			// This means the deployment was already deleted/finalized and there are
			// stale queue entries (or something edge triggering from a related
			// resource that got deleted async).
			r.Log.Info("deployment not found; reconciliation will be skipped", "request", req)
			return ctrl.Result{}, nil
		}
		// Error reading the object, so requeue the request.
		return ctrl.Result{}, fmt.Errorf("failed to get contour %q: %w", req, err)
	}

	// The deployment is safe to process, so ensure current state matches desired state.
	desired := deploy.ObjectMeta.DeletionTimestamp.IsZero()
	if desired {
		if err := r.ensureContour(ctx, contour); err != nil {
			switch e := err.(type) {
			case retryable.Error:
				r.Log.Error(e, "got retryable error; requeueing", "after", e.After())
				return ctrl.Result{RequeueAfter: e.After()}, nil
			default:
				return ctrl.Result{}, err
			}
		}
	} else {
		if err := r.ensureContourRemoved(ctx, contour); err != nil {
			switch e := err.(type) {
			case retryable.Error:
				r.Log.Error(e, "got retryable error; requeueing", "after", e.After())
				return ctrl.Result{RequeueAfter: e.After()}, nil
			default:
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Contour{}).
		Complete(r)
}

// otherContoursExist lists Contour objects in all namespaces, returning the list
// and true if any exist other than contour.
func (r *Reconciler) otherContoursExist(ctx context.Context, contour *operatorv1alpha1.Contour) (bool, *operatorv1alpha1.ContourList, error) {
	contours := &operatorv1alpha1.ContourList{}
	if err := r.Client.List(ctx, contours); err != nil {
		return false, nil, fmt.Errorf("failed to list contours: %w", err)
	}
	if len(contours.Items) == 0 || len(contours.Items) == 1 && contours.Items[0].Name == contour.Name {
		return false, nil, nil
	}
	return true, contours, nil
}

// otherContoursExistInSpecNs lists Contour objects in the same spec.namespace.name as contour,
// returning true if any exist.
func (r *Reconciler) otherContoursExistInSpecNs(ctx context.Context, contour *operatorv1alpha1.Contour) (bool, error) {
	exist, contours, err := r.otherContoursExist(ctx, contour)
	if err != nil {
		return false, err
	}
	if exist {
		for _, c := range contours.Items {
			if c.Spec.Namespace.Name == contour.Spec.Namespace.Name {
				return true, nil
			}
		}
	}
	return false, nil
}

// contourOwningSelector returns a label selector using "contour.operator.projectcontour.io/owning-contour"
// as the key and the contour name as the value.
func contourOwningSelector(contour *operatorv1alpha1.Contour) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			operatorv1alpha1.OwningContourLabel: contour.Name,
		},
	}
}

// contourDeploymentPodSelector returns a label selector using
// "contour.operator.projectcontour.io/deployment-contour" as the key
// and the contour name as the value.
func contourDeploymentPodSelector(contour *operatorv1alpha1.Contour) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			contourDeploymentLabel: contour.Name,
		},
	}
}

// envoyDaemonSetPodSelector returns a label selector using
// "contour.operator.projectcontour.io/daemonset-envoy" as the key and
// contour name as the value.
func envoyDaemonSetPodSelector(contour *operatorv1alpha1.Contour) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			envoyDaemonSetLabel: contour.Name,
		},
	}
}
