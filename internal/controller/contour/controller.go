/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package contour

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	retryable "github.com/projectcontour/contour-operator/internal/util/retryableerror"
	corev1 "k8s.io/api/core/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/api/errors"
	"github.com/projectcontour/contour-operator/internal/util/slice"

	operatorv1alpha1 "github.com/projectcontour/contour-operator/api/v1alpha1"
)

const (
	contourFinalizer string = "contour.operator.projectcontour.io/finalizer"

	defaultContourNamespace = "projectcontour"
)

// Config holds all the things necessary for the controller to run.
type Config struct {
	// Image is the name of the Contour container image.
	Image string
}

// Reconciler reconciles a Contour object.
type Reconciler struct {
	Config
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=operator.projectcontour.io,resources=contours,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.projectcontour.io,resources=contours/status,verbs=get;update;patch

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("contour", req.NamespacedName)

	r.Log.Info("reconciling", "request", req)

	// Only proceed if we can get the state of contour.
	contour := &operatorv1alpha1.Contour{}
	if err := r.Get(context.TODO(), req.NamespacedName, contour); err != nil {
		if errors.IsNotFound(err) {
			// This means the contour was already deleted/finalized and there are
			// stale queue entries (or something edge triggering from a related
			// resource that got deleted async).
			r.Log.Info("contour not found; reconciliation will be skipped", "request", req)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get contour %q: %v", req, err)
	}


	// If the contour is deleted, handle that and return early.
	if contour.DeletionTimestamp != nil {
		if err := r.ensureContourDeleted(contour); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to ensure contour deletion: %v", err)
		}
		r.Log.Info("contour was successfully deleted", "namespace", contour.Namespace, "name", contour.Name)
		return ctrl.Result{}, nil
	}

	// The contour is safe to process, so ensure current state matches desired state.
	if err := r.ensureContour(contour); err != nil {
		switch e := err.(type) {
		case retryable.Error:
			r.Log.Error(e, "got retryable error; requeueing", "after", e.After())
			return ctrl.Result{RequeueAfter: e.After()}, nil
		default:
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Contour{}).
		Complete(r)
}

// ensureContourDeleted tries to delete contour, and if successful, will remove
// the finalizer.
func (r *Reconciler) ensureContourDeleted(contour *operatorv1alpha1.Contour) error {
	errs := []error{}

	if err := r.ensureDeploymentDeleted(contour); err != nil {
		errs = append(errs, fmt.Errorf("failed to delete deployment for contour %s/%s: %v",
			contour.Namespace, contour.Name, err))
	}

	if len(errs) == 0 {
		// Remove the contour finalizer.
		if slice.ContainsString(contour.Finalizers, contourFinalizer) {
			updated := contour.DeepCopy()
			updated.Finalizers = slice.RemoveString(updated.Finalizers, manifests.IngressControllerFinalizer)
			if err := r.Update(context.TODO(), updated); err != nil {
				errs = append(errs, fmt.Errorf("failed to remove finalizer from contour %s/%s: %v",
					contour.Namespace, contour.Name, err))
			}
		}
	}

	return utilerrors.NewAggregate(errs)
}

// ensureContour ensures all necessary resources exist for the given contour.
// Any error values are collected into either a retryable.Error value, if any
// of the error values are retryable, or else an Aggregate error value.
func (r *Reconciler) ensureContour(contour *operatorv1alpha1.Contour) error {
	// Before doing anything at all with the contour, ensure it has a finalizer
	// so we can clean up later.
	if !slice.ContainsString(contour.Finalizers, contourFinalizer) {
		updated := contour.DeepCopy()
		updated.Finalizers = append(updated.Finalizers, contourFinalizer)
		if err := r.Update(context.TODO(), updated); err != nil {
			return fmt.Errorf("failed to update finalizers: %v", err)
		}
		if err := r.Get(context.TODO(), types.NamespacedName{Namespace: updated.Namespace, Name: updated.Name}, updated); err != nil {
			return fmt.Errorf("failed to get ingresscontroller: %v", err)
		}
		contour = updated
	}

	ns, err := r.ensureNamespace(defaultContourNamespace)
	if err != nil {
		return fmt.Errorf("failed to ensure namespace: %v", err)
	}

	if err := r.ensureRBAC(ns); err != nil {
		return fmt.Errorf("failed to ensure rbac: %v", err)
	}

	var errs []error
	if err := r.ensureConfigMap(); err != nil {
		errs = append(errs, err)
	}

	haveDepl, deployment, err := r.ensureDeployment(contour)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to ensure deployment: %v", err))
		return utilerrors.NewAggregate(errs)
	} else if !haveDepl {
		errs = append(errs, fmt.Errorf("failed to get contour deployment %s/%s", contour.Namespace, contour.Name))
		return utilerrors.NewAggregate(errs)
	}

	trueVar := true
	deploymentRef := metav1.OwnerReference{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Name:       deployment.Name,
		UID:        deployment.UID,
		Controller: &trueVar,
	}

	var lbService *corev1.Service
	if haveLB, lb, err := r.ensureLoadBalancerService(contour, deploymentRef); err != nil {
		errs = append(errs, fmt.Errorf("failed to ensure load balancer service for %s/%s: %v",
			contour.Namespace, contour.Name, err))
	}

	if internalSvc, err := r.ensureInternalContourService(contour, deploymentRef); err != nil {
		errs = append(errs, fmt.Errorf("failed to create internal service for contour %s/%s: %v",
			contour.Namespace, contour.Name, err))
	}

	errs = append(errs, r.syncContourStatus(contour, deployment, lbService, internalSvc))

	return retryable.NewMaybeRetryableAggregate(errs)
}
