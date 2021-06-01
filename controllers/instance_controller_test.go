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

package controllers

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"time"

	firewall "repo-url/api/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("crd controller", func() {
	const (
		// duration = time.Second * 10
		interval = time.Second * 2
		timeout  = time.Second * 30
	)
	source := &firewall.ClusterwideNetworkPolicy{}

	Context("syncing the change between source and destination cluster", func() {
		It("should create the same instance in destination cluster upon a new instance in source cluster", func() {
			Expect(createInstance(sourceClusterClient, filepath.Join("..", "test", "cwnp-sample.yaml"), source)).Should(Succeed())
			Eventually(func() bool {
				dest := &firewall.ClusterwideNetworkPolicy{}
				if err := destClusterClient.Get(newCtx(), client.ObjectKeyFromObject(source), dest); err != nil {
					log.Print(err)
					return false
				}
				return reflect.DeepEqual(source.Spec, dest.Spec)
			}, timeout, interval).Should(BeTrue())
		})

		It("should update the instance in destination cluster upon a changed instance in source cluster", func() {
			patch := client.MergeFrom(source.DeepCopy())
			newPort := intstr.FromInt(9999)
			source.Spec.Egress[0].Ports[0].Port = &newPort
			source.Spec.Egress[0].Ports[1].Port = &newPort
			Expect(sourceClusterClient.Patch(newCtx(), source, patch)).Should(Succeed())

			Eventually(func() bool {
				dest := &firewall.ClusterwideNetworkPolicy{}
				if err := destClusterClient.Get(newCtx(), client.ObjectKeyFromObject(source), dest); err != nil {
					log.Print(err)
					return false
				}
				return reflect.DeepEqual(source.Spec, dest.Spec)
			}, timeout, interval).Should(BeTrue())
		})

		It("should delete the instance in destination cluster upon deletion of the instance in source cluster", func() {
			Expect(sourceClusterClient.Delete(newCtx(), source)).Should(Succeed())
			Eventually(func() bool {
				return destClusterClient.Get(newCtx(), client.ObjectKeyFromObject(source), &firewall.ClusterwideNetworkPolicy{}) != nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})

func createInstance(k8sClient client.Client, filePath string, instance *firewall.ClusterwideNetworkPolicy) error {
	defer GinkgoRecover()

	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", filePath, err)
	}

	if err := yaml.Unmarshal(bytes, instance); err != nil {
		return fmt.Errorf("unmarshalling yaml: %w", err)
	}

	if err := ensureNamespace(newCtx(), sourceClusterClient, instance.Namespace); err != nil {
		return fmt.Errorf("ensuring namespace %s: %w", instance.Namespace, err)
	}

	log.Printf("%#v", instance)

	return sourceClusterClient.Create(newCtx(), instance)
}
