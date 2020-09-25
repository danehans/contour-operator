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
	"github.com/projectcontour/contour-operator/util/slice"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	finalizerSuffix = "-finalizer"
	testFinalizer   = contourFinalizer + finalizerSuffix
)

var _ = Describe("Run controller", func() {

	Context("When creating a contour", func() {
		It("Finalizer should be set", func() {
			By("By creating a contour")

			key := types.NamespacedName{
				Name:      contourName + finalizerSuffix,
				Namespace: operatorNamespace,
			}

			// Create a contour with ns delete set
			created := &operatorv1alpha1.Contour{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: key.Namespace,
					Name:      key.Name,
				},
				Spec: operatorv1alpha1.ContourSpec{
					Namespace: operatorv1alpha1.NamespaceSpec{
						RemoveOnDeletion: true,
					},
				},
			}
			Expect(k8sClient.Create(ctx, created)).Should(Succeed())

			// Check contour finalizer
			By("By finalizing the contour")
			Eventually(func() bool {
				f := &operatorv1alpha1.Contour{}
				k8sClient.Get(ctx, key, f)
				return slice.ContainsString(f.Finalizers, contourFinalizer)
			}, timeout, interval).Should(Equal(true))

			// Update finalizers
			updated := &operatorv1alpha1.Contour{}
			Expect(k8sClient.Get(ctx, key, updated)).Should(Succeed())
			updated.Finalizers = append(updated.Finalizers, testFinalizer)
			Expect(k8sClient.Update(ctx, updated)).Should(Succeed())

			// Check contour finalizer still exists
			By("Expecting the contour and test finalizers")
			Eventually(func() bool {
				f := &operatorv1alpha1.Contour{}
				k8sClient.Get(ctx, key, f)
				return slice.ContainsString(f.Finalizers, contourFinalizer) &&
					slice.ContainsString(f.Finalizers, testFinalizer)
			}, timeout, interval).Should(Equal(true))

			// Remove test finalizer
			updated = &operatorv1alpha1.Contour{}
			Expect(k8sClient.Get(ctx, key, updated)).Should(Succeed())
			updated.Finalizers = slice.RemoveString(updated.Finalizers, testFinalizer)
			Expect(k8sClient.Update(ctx, updated)).Should(Succeed())

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
