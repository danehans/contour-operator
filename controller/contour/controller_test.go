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
	operatorv1alpha1 "github.com/projectcontour/contour-operator/api/v1alpha1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Run controller", func() {

	Context("When creating a contour", func() {
		It("Defaults should be set", func() {
			By("By creating a contour with a nil spec")

			key := types.NamespacedName{
				Name:      contourName,
				Namespace: operatorNamespace,
			}

			// Create a contour
			created := &operatorv1alpha1.Contour{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: key.Namespace,
					Name:      key.Name,
				},
			}
			Expect(k8sClient.Create(ctx, created)).Should(Succeed())

			// Check replicas
			Eventually(func() int32 {
				f := &operatorv1alpha1.Contour{}
				k8sClient.Get(ctx, key, f)
				return f.Spec.Replicas
			}, timeout, interval).Should(Equal(int32(defaultReplicas)))

			// Check namespace name
			Eventually(func() string {
				f := &operatorv1alpha1.Contour{}
				k8sClient.Get(ctx, key, f)
				return f.Spec.Namespace.Name
			}, timeout, interval).Should(Equal(defaultNamespace))

			// Check namespace remove on deletion
			Eventually(func() bool {
				f := &operatorv1alpha1.Contour{}
				k8sClient.Get(ctx, key, f)
				return f.Spec.Namespace.RemoveOnDeletion
			}, timeout, interval).Should(Equal(defaultRemoveNs))

			// Update the contour
			updated := &operatorv1alpha1.Contour{}
			Expect(k8sClient.Get(ctx, key, updated)).Should(Succeed())
			updated.Spec.Replicas = int32(1)
			testUpdatedNs := defaultNamespace + "-updated"
			updated.Spec.Namespace.Name = testUpdatedNs
			updated.Spec.Namespace.RemoveOnDeletion = true
			Expect(k8sClient.Update(ctx, updated)).Should(Succeed())

			By("Expecting 1 replica")
			Eventually(func() int32 {
				f := &operatorv1alpha1.Contour{}
				k8sClient.Get(ctx, key, f)
				return f.Spec.Replicas
			}, timeout, interval).Should(Equal(int32(1)))

			By("Expecting spec.namespace.name updated namespace")
			Eventually(func() string {
				f := &operatorv1alpha1.Contour{}
				k8sClient.Get(ctx, key, f)
				return f.Spec.Namespace.Name
			}, timeout, interval).Should(Equal(testUpdatedNs))

			By("Expecting remove on deletion")
			Eventually(func() bool {
				f := &operatorv1alpha1.Contour{}
				k8sClient.Get(ctx, key, f)
				return f.Spec.Namespace.RemoveOnDeletion
			}, timeout, interval).Should(Equal(true))

			// Delete the contour
			By("Expecting to delete contour successfully")
			Eventually(func() error {
				f := &operatorv1alpha1.Contour{}
				k8sClient.Get(ctx, key, f)
				return k8sClient.Delete(ctx, f)
			}, timeout, interval).Should(Succeed())

			By("Expecting contour delete to finish")
			Eventually(func() error {
				f := &operatorv1alpha1.Contour{}
				return k8sClient.Get(ctx, key, f)
			}, timeout, interval).ShouldNot(Succeed())
		})
	})
})
