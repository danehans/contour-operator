package contour

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// ensureNamespace ensures the namespace for the provided name exists.
func (r *Reconciler) ensureNamespace(desired bool, name string) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if err := r.Get(context.TODO(), types.NamespacedName{Name: ns.Name}, ns); err != nil {
		if !errors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to get namespace %s: %v", ns.Name, err)
		}
		if desired {
			if err := r.Create(context.TODO(), ns); err != nil {
				return nil, fmt.Errorf("failed to create namespace %s: %v", ns.Name, err)
			}
			r.Log.Info("created namespace", "name", ns.Name)
		} else {
			if err := r.Delete(context.TODO(), ns); err != nil {
				return nil, fmt.Errorf("failed to delete namespace %s: %v", ns.Name, err)
			}
			r.Log.Info("deleted namespace", "name", ns.Name)
		}
	}
	return ns, nil
}
