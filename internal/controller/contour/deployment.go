package contour

import (
	"context"
	operatorv1alpha1 "github.com/projectcontour/contour-operator/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"

	"k8s.io/apimachinery/pkg/api/errors"
)

// ensureDeploymentDeleted ensures the Contour deployment is deleted.
func (r *Reconciler) ensureDeploymentDeleted(contour *operatorv1alpha1.Contour) error {
	deployment := &appsv1.Deployment{}
	deployment.Name = contour.Name
	deployment.Namespace = contour.Namespace
	if err := r.Delete(context.TODO(), deployment); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	r.Log.Info("deleted deployment", "namespace", deployment.Namespace, "name", deployment.Name)
	return nil
}
