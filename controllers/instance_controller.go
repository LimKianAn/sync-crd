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
	"fmt"
	api "repo-url/api/v1"

	"github.com/go-logr/logr"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	cr "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Don't put the follwing two types under a same `type` keyword. This way it's easier for build-time `sed`.
type CRD = api.ClusterwideNetworkPolicy
type CRDReconciler struct {
	Source client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Dest   client.Client
}

func (r *CRDReconciler) Reconcile(ctx context.Context, req cr.Request) (cr.Result, error) {
	log := r.Log.WithValues("crd", req.NamespacedName)

	source := &CRD{}
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

	dest := &CRD{}
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

func crdWithObjKey(objKey *types.NamespacedName) *CRD {
	c := &CRD{}
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
