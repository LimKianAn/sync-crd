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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	api "repo-url/api/v1"

	"github.com/go-logr/logr"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/clientcmd"
	cr "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type crd = api.Instance

var (
	flags = struct {
		metricsAddr, namespace, source, destination string
		enableLeaderElection                        bool
	}{}
	log           = cr.Log.WithName("setup")
	scheme        = runtime.NewScheme()
	schemeBuilder = runtime.SchemeBuilder{
		core.AddToScheme,
		api.AddToScheme,
	}
)

type (
	extendedMgr struct {
		manager.Manager
	}

	instanceReconciler struct {
		Source client.Client
		Log    logr.Logger
		Scheme *runtime.Scheme
		Dest   client.Client
	}
)

func init() {
	utilruntime.Must(schemeBuilder.AddToScheme(scheme))
}

func main() {
	parseflags()
	cr.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := newMgr()
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := mgr.registerCRDController(); err != nil {
		log.Error(err, "unable to create a new controller")
		os.Exit(1)
	}

	if err := mgr.Start(context.Background()); err != nil {
		log.Error(err, "unable to start a manager")
		os.Exit(1)
	}
}

func newMgr() (*extendedMgr, error) {
	SourceConfig, err := clientcmd.BuildConfigFromFlags("", flags.source)
	if err != nil {
		log.Error(err, "unable to build config from flags")
		return nil, err
	}

	mgr, err := cr.NewManager(SourceConfig, cr.Options{
		Scheme:             scheme,
		MetricsBindAddress: flags.metricsAddr,
		Port:               9443,
		LeaderElection:     flags.enableLeaderElection,
		LeaderElectionID:   "syncrd",
		Namespace:          flags.namespace,
	})
	if err != nil {
		log.Error(err, "unable to create a new manager")
		return nil, err
	}

	return &extendedMgr{Manager: mgr}, nil
}

func (m *extendedMgr) registerCRDController() error {
	destinationConfig, err := clientcmd.BuildConfigFromFlags("", flags.destination)
	if err != nil {
		return fmt.Errorf("unable to build config from flags: %w", err)
	}

	destK8sClient, err := client.New(destinationConfig, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("unable to create a new k8s client: %w", err)
	}

	return cr.NewControllerManagedBy(m).
		For(&crd{}).
		Complete(&instanceReconciler{
			Source: m.GetClient(),
			Log:    cr.Log.WithName("controllers").WithName("CRD"),
			Scheme: m.GetScheme(),
			Dest:   destK8sClient,
		})
}

func (r *instanceReconciler) Reconcile(ctx context.Context, req cr.Request) (cr.Result, error) {
	log := r.Log.WithValues("instance", req.NamespacedName)

	source := &crd{}
	if err := r.Source.Get(ctx, req.NamespacedName, source); err != nil {
		if !errors.IsNotFound(err) {
			return cr.Result{}, err
		}

		if err := r.Dest.Delete(ctx, crdWithObjKey(&req.NamespacedName)); err != nil {
			return cr.Result{}, fmt.Errorf("failed to delete the destination instance: %w", err)
		}
		log.Info("destination instance deleted")

		return cr.Result{}, nil
	}
	log.Info("source instance fetched")

	if err := ensureNamespace(ctx, r.Dest, req.NamespacedName.Namespace); err != nil {
		return cr.Result{}, fmt.Errorf("failed to ensure namespace %s: %w", source.Namespace, err)
	}
	log.Info("namespace in destination cluster ensured")

	dest := &crd{}
	dest.Namespace = source.Namespace
	dest.Name = source.Name
	_, err := controllerutil.CreateOrPatch(ctx, r.Dest, dest, func() error {
		dest.Spec = source.Spec
		return nil
	})
	if err != nil {
		return cr.Result{}, fmt.Errorf("failed to reconcile the destination instance: %w", err)
	}
	log.Info("destination instance reconciled")

	return cr.Result{}, nil
}

func crdWithObjKey(objKey *types.NamespacedName) *crd {
	c := &crd{}
	c.Namespace = objKey.Namespace
	c.Name = objKey.Name
	return c
}

func ensureNamespace(ctx context.Context, k8sClient client.Client, namespace string) error {
	nsObj := &core.Namespace{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: namespace}, nsObj); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to fetch namespace %s: %w", namespace, err)
		}

		nsObj.Name = namespace
		if err := k8sClient.Create(ctx, nsObj); err != nil {
			return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
		}
	}
	return nil
}

func parseflags() {
	flag.StringVar(&flags.metricsAddr, "metrics-addr", ":9999", "The address the metric endpoint binds to")
	flag.StringVar(&flags.namespace, "namespace", "", "The watched namespace in the source cluster")
	flag.StringVar(&flags.source, "source", "/tmp/source", "The path to the kubeconfig of the source cluster")
	flag.StringVar(&flags.destination, "dest", "/tmp/dest", "The path to the kubeconfig of the destination cluster")
	flag.BoolVar(&flags.enableLeaderElection, "enable-leader-election", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.Parse()
}
