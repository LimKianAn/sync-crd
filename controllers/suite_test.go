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
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	cr "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	firewall "repo-url/api/v1"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

var (
	sourceClusterCfg     *rest.Config
	sourceClusterClient  client.Client
	sourceClusterTestEnv *envtest.Environment

	destClusterCfg     *rest.Config
	destClusterClient  client.Client
	destClusterTestEnv *envtest.Environment

	// externalYAMLDir     = filepath.Join("..", "external")
	// externalYAMLDirTest = filepath.Join(externalYAMLDir, "test")

	// instance = &firewall.ClusterwideNetworkPolicy{}
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	defer close(done)

	By("bootstrapping test environment")

	// Create test env for source cluster and start it
	sourceClusterTestEnv = &envtest.Environment{
		// Path to CRD for testing
		CRDDirectoryPaths: []string{filepath.Join("..", "test", "crd")},
	}
	sourceClusterCfg = startTestEnv(sourceClusterTestEnv)

	scheme := newScheme()
	sourceClusterMgr, err := cr.NewManager(sourceClusterCfg, cr.Options{MetricsBindAddress: "0", Scheme: scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(sourceClusterMgr).ToNot(BeNil())
	sourceClusterClient = sourceClusterMgr.GetClient()

	// Create test env for destination cluster and start it
	destClusterTestEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "test", "crd")},
	}
	destClusterCfg = startTestEnv(destClusterTestEnv)

	destClusterClient, err = client.New(destClusterCfg, client.Options{Scheme: scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(destClusterClient).ToNot(BeNil())

	cr.NewControllerManagedBy(sourceClusterMgr).
		For(&CRD{}).
		Complete(&CRDReconciler{
			Source: sourceClusterMgr.GetClient(),
			Log:    cr.Log.WithName("controllers").WithName("crd"),
			Scheme: sourceClusterMgr.GetScheme(),
			Dest:   destClusterClient,
		})
	go startMgr(sourceClusterMgr)

	cr.SetLogger(zap.New(zap.UseDevMode(true)))

}, 1000)

var _ = AfterSuite(func() {
	By("tearing down the test environment")

	err := sourceClusterTestEnv.Stop()
	Expect(err).ToNot(HaveOccurred())

	err = destClusterTestEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func newCtx() context.Context {
	return context.Background()
}

func newScheme() *runtime.Scheme {
	defer GinkgoRecover()

	scheme := runtime.NewScheme()
	Expect(core.AddToScheme(scheme)).Should(Succeed())
	Expect(apiextensions.AddToScheme(scheme)).Should(Succeed())
	Expect(firewall.AddToScheme(scheme)).Should(Succeed())

	return scheme
}

func startMgr(mgr manager.Manager) {
	defer GinkgoRecover()
	Expect(mgr.Start(newCtx())).Should(Succeed())
}

func startTestEnv(env *envtest.Environment) *rest.Config {
	defer GinkgoRecover()

	cfg, err := env.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	return cfg
}
