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

package equality

import (
	operatorv1alpha1 "github.com/projectcontour/contour-operator/api/v1alpha1"

	contourv1alpha1 "github.com/projectcontour/contour/apis/projectcontour/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
)

// DaemonsetConfigChanged checks if current and expected DaemonSet match,
// and if not, returns the updated DaemonSet resource.
func DaemonsetConfigChanged(current, expected *appsv1.DaemonSet) (*appsv1.DaemonSet, bool) {
	changed := false
	updated := current.DeepCopy()

	if !apiequality.Semantic.DeepEqual(current.Labels, expected.Labels) {
		changed = true
		updated.Labels = expected.Labels

	}

	if !apiequality.Semantic.DeepEqual(current.Spec, expected.Spec) {
		changed = true
		updated.Spec = expected.Spec
	}

	if !changed {
		return nil, false
	}

	return updated, true
}

// DaemonSetSelectorsDiffer checks if the current and expected DaemonSet selectors differ.
func DaemonSetSelectorsDiffer(current, expected *appsv1.DaemonSet) bool {
	return !apiequality.Semantic.DeepEqual(current.Spec.Selector, expected.Spec.Selector)
}

// JobConfigChanged checks if the current and expected Job match and if not,
// returns true and the expected job.
func JobConfigChanged(current, expected *batchv1.Job) (*batchv1.Job, bool) {
	changed := false
	updated := current.DeepCopy()

	if !apiequality.Semantic.DeepEqual(current.Labels, expected.Labels) {
		updated = expected
		changed = true
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.Parallelism, expected.Spec.Parallelism) {
		updated = expected
		changed = true
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.BackoffLimit, expected.Spec.BackoffLimit) {
		updated = expected
		changed = true
	}

	// The completions field is immutable, so no need to compare. Ignore job-generated
	// labels and only check the presence of the contour owning labels.
	if current.Spec.Template.Labels != nil {
		_, nameFound := current.Spec.Template.Labels[operatorv1alpha1.OwningContourNameLabel]
		_, nsFound := current.Spec.Template.Labels[operatorv1alpha1.OwningContourNsLabel]
		if !nameFound || !nsFound {
			updated = expected
			changed = true
		}
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.Template.Spec, expected.Spec.Template.Spec) {
		updated = expected
		changed = true
	}

	if !changed {
		return nil, false
	}

	return updated, true
}

// DeploymentConfigChanged checks if the current and expected Deployment match
// and if not, returns true and the expected Deployment.
func DeploymentConfigChanged(current, expected *appsv1.Deployment) (*appsv1.Deployment, bool) {
	changed := false
	updated := current.DeepCopy()

	if !apiequality.Semantic.DeepEqual(current.Labels, expected.Labels) {
		updated = expected
		changed = true
	}

	if !apiequality.Semantic.DeepEqual(current.Spec, expected.Spec) {
		updated = expected
		changed = true
	}

	if !changed {
		return nil, false
	}

	return updated, true
}

// DeploymentSelectorsDiffer checks if the current and expected Deployment selectors differ.
func DeploymentSelectorsDiffer(current, expected *appsv1.Deployment) bool {
	return !apiequality.Semantic.DeepEqual(current.Spec.Selector, expected.Spec.Selector)
}

// ClusterIPServiceChanged checks if the spec of current and expected match and if not,
// returns true and the expected Service resource. The cluster IP is not compared
// as it's assumed to be dynamically assigned.
func ClusterIPServiceChanged(current, expected *corev1.Service) (*corev1.Service, bool) {
	changed := false
	updated := current.DeepCopy()

	// Spec can't simply be matched since clusterIP is being dynamically assigned.
	if len(current.Spec.Ports) != len(expected.Spec.Ports) {
		updated.Spec.Ports = expected.Spec.Ports
		changed = true
	} else {
		if !apiequality.Semantic.DeepEqual(current.Spec.Ports, expected.Spec.Ports) {
			updated.Spec.Ports = expected.Spec.Ports
			changed = true
		}
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.Selector, expected.Spec.Selector) {
		updated.Spec.Selector = expected.Spec.Selector
		changed = true
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.SessionAffinity, expected.Spec.SessionAffinity) {
		updated.Spec.SessionAffinity = expected.Spec.SessionAffinity
		changed = true
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.Type, expected.Spec.Type) {
		updated.Spec.Type = expected.Spec.Type
		changed = true
	}

	if !changed {
		return nil, false
	}

	return updated, true
}

// ContourStatusChanged checks if current and expected match and if not,
// returns true.
func ContourStatusChanged(current, expected operatorv1alpha1.ContourStatus) bool {
	if current.AvailableContours != expected.AvailableContours {
		return true
	}

	if current.AvailableEnvoys != expected.AvailableEnvoys {
		return true
	}

	if !apiequality.Semantic.DeepEqual(current.Conditions, expected.Conditions) {
		return true
	}

	return false
}

// EnvoyChanged checks if current and expected match and if not,
// returns true.
func EnvoyChanged(current, expected *contourv1alpha1.Envoy) bool {
	if !apiequality.Semantic.DeepEqual(current.Labels, expected.Labels) {
		return true
	}

	if !apiequality.Semantic.DeepEqual(current.Spec, expected.Spec) {
		return true
	}

	return false
}

// NamespaceConfigChanged checks if the current and expected Namespace match
// and if not, returns true and the expected Namespace.
func NamespaceConfigChanged(current, expected *corev1.Namespace) (*corev1.Namespace, bool) {
	changed := false
	updated := current.DeepCopy()

	if !apiequality.Semantic.DeepEqual(current.Labels, expected.Labels) {
		updated = expected
		changed = true
	}

	if !changed {
		return nil, false
	}

	return updated, true
}

// ServiceAccountConfigChanged checks if the current and expected ServiceAccount
// match and if not, returns true and the expected ServiceAccount.
func ServiceAccountConfigChanged(current, expected *corev1.ServiceAccount) (*corev1.ServiceAccount, bool) {
	changed := false
	updated := current.DeepCopy()

	if !apiequality.Semantic.DeepEqual(current.Labels, expected.Labels) {
		updated = expected
		changed = true
	}

	if !changed {
		return nil, false
	}

	return updated, true
}

// ClusterRoleConfigChanged checks if the current and expected ClusterRole
// match and if not, returns true and the expected ClusterRole.
func ClusterRoleConfigChanged(current, expected *rbacv1.ClusterRole) (*rbacv1.ClusterRole, bool) {
	changed := false
	updated := current.DeepCopy()

	if !apiequality.Semantic.DeepEqual(current.Labels, expected.Labels) {
		changed = true
		updated.Labels = expected.Labels
	}

	if !apiequality.Semantic.DeepEqual(current.Rules, expected.Rules) {
		changed = true
		updated.Rules = expected.Rules
	}

	if !changed {
		return nil, false
	}

	return updated, true
}

// ClusterRoleBindingConfigChanged checks if the current and expected ClusterRoleBinding
// match and if not, returns true and the expected ClusterRoleBinding.
func ClusterRoleBindingConfigChanged(current, expected *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, bool) {
	changed := false
	updated := current.DeepCopy()

	if !apiequality.Semantic.DeepEqual(current.Labels, expected.Labels) {
		changed = true
		updated.Labels = expected.Labels

	}

	if !apiequality.Semantic.DeepEqual(current.Subjects, expected.Subjects) {
		changed = true
		updated.Subjects = expected.Subjects
	}

	if !apiequality.Semantic.DeepEqual(current.RoleRef, expected.RoleRef) {
		changed = true
		updated.RoleRef = expected.RoleRef
	}

	if !changed {
		return nil, false
	}

	return updated, true
}

// RoleConfigChanged checks if the current and expected Role match
// and if not, returns true and the expected Role.
func RoleConfigChanged(current, expected *rbacv1.Role) (*rbacv1.Role, bool) {
	changed := false
	updated := current.DeepCopy()

	if !apiequality.Semantic.DeepEqual(current.Labels, expected.Labels) {
		changed = true
		updated.Labels = expected.Labels
	}

	if !apiequality.Semantic.DeepEqual(current.Rules, expected.Rules) {
		changed = true
		updated.Rules = expected.Rules
	}

	if !changed {
		return nil, false
	}

	return updated, true
}

// RoleBindingConfigChanged checks if the current and expected RoleBinding
// match and if not, returns true and the expected RoleBinding.
func RoleBindingConfigChanged(current, expected *rbacv1.RoleBinding) (*rbacv1.RoleBinding, bool) {
	changed := false
	updated := current.DeepCopy()

	if !apiequality.Semantic.DeepEqual(current.Labels, expected.Labels) {
		changed = true
		updated.Labels = expected.Labels

	}

	if !apiequality.Semantic.DeepEqual(current.Subjects, expected.Subjects) {
		changed = true
		updated.Subjects = expected.Subjects
	}

	if !apiequality.Semantic.DeepEqual(current.RoleRef, expected.RoleRef) {
		changed = true
		updated.RoleRef = expected.RoleRef
	}

	if !changed {
		return nil, false
	}

	return updated, true
}
