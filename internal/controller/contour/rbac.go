package contour

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	defaultContourServiceAccount = "contour"
	defaultEnvoyServiceAccount   = "envoy"
)

func (r *Reconciler) ensureRBAC(exists bool, ns *corev1.Namespace) error {
	contour := types.NamespacedName{
		Namespace: ns.Name,
		Name:      defaultContourServiceAccount,
	}
	envoy := types.NamespacedName{
		Namespace: ns.Name,
		Name:      defaultEnvoyServiceAccount,
	}
	names := []types.NamespacedName{contour,envoy}
	if err := r.ensureServiceAccounts(exists, names); err != nil {
		return err
	}
	if err := r.ensureClusterRoleBinding(exists, defaultContourServiceAccount); err != nil {
		return err
	}
	if err := r.ensureClusterRoles(exists, defaultContourServiceAccount); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) ensureServiceAccounts(exists bool, nsNames []types.NamespacedName) error {
	for _, name := range nsNames {
		sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name.Name, Namespace: name.Namespace}}
		if err := r.Get(context.TODO(), name, sa); err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("failed to get service account %s/%s: %v", sa.Namespace, sa.Name, err)
			}
			if exists {
				if err := r.Create(context.TODO(), sa); err != nil {
					return fmt.Errorf("failed to create service account %s/%s: %v", sa.Namespace, sa.Name, err)
				}
				r.Log.Info("created service account", "namespace", sa.Namespace, "name", sa.Name)
			} else {
				if err := r.Delete(context.TODO(), sa); err != nil {
					return fmt.Errorf("failed to delete service account %s/%s: %v", sa.Namespace, sa.Name, err)
				}
				r.Log.Info("created service account", "namespace", sa.Namespace, "name", sa.Name)
			}
		}
	}
	return nil
}

func (r *Reconciler) ensureClusterRoleBinding(name string) error {
	crb := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if err := r.Get(context.TODO(), types.NamespacedName{Name: crb.Name}, crb); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get cluster role binding %s: %v", crb.Name, err)
		}
		if err := r.Create(context.TODO(), crb); err != nil {
			return fmt.Errorf("failed to create cluster role binding %s: %v", crb.Name, err)
		}
		r.Log.Info("created cluster role binding", "name", crb.Name)
	}
}

func (r *Reconciler) ensureClusterRoles(name string) error {
	desired := desiredClusterRole(name)

	haveCR, current, err := r.currentClusterRole(name)
	if err != nil {
		return err
	}

	switch {
	case !wantCR && !haveCR:
		return false, nil, nil
	case !wantCR && haveCR:
		if err := r.Delete(context.TODO(), current); err != nil {
			if !errors.IsNotFound(err) {
				return true, current, fmt.Errorf("failed to delete cluster role: %v", err)
			}
		} else {
			r.Log.Info("deleted cluster role", "current", current)
		}
		return false, nil, nil
	case wantCR && !haveCR:
		if err := r.Create(context.TODO(), desired); err != nil {
			return false, nil, fmt.Errorf("failed to create cluster role: %v", err)
		}
		log.Info("created cluster role", "desired", desired)
		return r.currentClusterRole()
	case wantCR && haveCR:
		if updated, err := r.updateClusterRole(current, desired); err != nil {
			return true, current, fmt.Errorf("failed to update cluster role: %v", err)
		} else if updated {
			log.Info("updated cluster role", "desired", desired)
			return r.currentClusterRole()
		}
	}
}

func desiredClusterRole(name string) *rbacv1.ClusterRole {
	groupAll := []string{""}
	groupNet := []string{"networking.k8s.io"}
	groupContour := []string{"projectcontour.io"}
	verbCGU := []string{"create", "get", "update"}
	verbGLW := []string{"get", "list", "watch"}

	cfgMap := rbacv1.PolicyRule{
		Verbs:     verbCGU,
		APIGroups: groupAll,
		Resources: []string{"configmaps"},
	}
	endPt := rbacv1.PolicyRule{
		Verbs:     verbGLW,
		APIGroups: groupAll,
		Resources: []string{"endpoints"},
	}
	secret := rbacv1.PolicyRule{
		Verbs:     verbGLW,
		APIGroups: groupAll,
		Resources: []string{"secrets"},
	}
	svc := rbacv1.PolicyRule{
		Verbs:     verbGLW,
		APIGroups: groupAll,
		Resources: []string{"services"},
	}
	svcAPI := rbacv1.PolicyRule{
		Verbs:     verbGLW,
		APIGroups: groupNet,
		// TODO [danehans] Update roles when v1alpha1 Service APIs are released.
		Resources:       []string{"gatewayclasses", "gateways", "httproutes", "tcproutes"},
	}
	ing := rbacv1.PolicyRule{
		Verbs:     verbGLW,
		APIGroups: groupNet,
		Resources:       []string{"ingresses"},
	}
	ingStatus := rbacv1.PolicyRule{
		Verbs:     verbCGU,
		APIGroups: groupNet,
		Resources:       []string{"ingresses/status"},
	}
	cntr := rbacv1.PolicyRule{
		Verbs:     verbGLW,
		APIGroups: groupContour,
		Resources:       []string{"httpproxies", "tlscertificatedelegations"},
	}
	cntrStatus := rbacv1.PolicyRule{
		Verbs:     verbCGU,
		APIGroups: groupContour,
		Resources:       []string{"httpproxies/status"},
	}

	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: []rbacv1.PolicyRule{cfgMap, endPt, secret, svc, svcAPI, ing, ingStatus, cntr, cntrStatus},
	}
}

func (r *Reconciler) currentClusterRole(name string) (bool, error) {
	cr := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if err := r.Get(context.TODO(), types.NamespacedName{Name: cr.Name}, cr); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, cr
}

func (r *reconciler) updateClusterRole(current, desired *rbacv1.ClusterRole) (bool, error) {
	if reflect.DeepEqual(current.Rules, desired.Rules) {
		return false, nil
	}
	updated := current.DeepCopy()
	updated.Rules = desired.Rules
	if err := r.client.Update(context.TODO(), updated); err != nil {
		if errors.IsAlreadyExists(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
