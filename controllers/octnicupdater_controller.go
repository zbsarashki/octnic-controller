/*
Copyright 2023 tbc project.

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
	"io/ioutil"
	"regexp"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/controller-runtime/pkg/log"

	acclrv1beta1 "github.com/zbsarashk/OctNic/api/v1beta1"
)

// OctNicUpdaterReconciler reconciles a OctNicUpdater object
type OctNicUpdaterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=acclr.github.com,resources=octnicupdaters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=acclr.github.com,resources=octnicupdaters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=acclr.github.com,resources=octnicupdaters/finalizers,verbs=update

//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OctNicUpdater object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
func (r *OctNicUpdaterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	mctx := &acclrv1beta1.OctNicUpdater{}
	if err := r.Get(ctx, req.NamespacedName, mctx); err != nil {
		fmt.Printf("unable to fetch UpdateJob: %s\n", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}


	/* Start driver daemonSet if not already started */
	driverSet := &appsv1.DaemonSet{}
	err := r.Get(context.TODO(),
			types.NamespacedName{
			Name: mctx.Spec.Acclr + "-driver", Namespace: mctx.Namespace},
			driverSet)
	if errors.IsNotFound(err) {
		/* Start daemonSet */
		fmt.Printf("err %s\n", err)
		p := appsv1.DaemonSet{}
		byf, err := ioutil.ReadFile("/manifests/drv-daemon/" +
				mctx.Spec.Acclr + "-drv.yaml")
		if err != nil {
			fmt.Printf("Manifests not found: %s\n", err)
			return ctrl.Result{}, nil
		}
		yamlutil.Unmarshal(byf, &p)
		err = ctrl.SetControllerReference(mctx, &p, r.Scheme)
		if err != nil {
			fmt.Printf("Failed to set controller reference: %s\n", err)
			return ctrl.Result{}, nil
		}
		err = r.Create(ctx, &p)
		if err != nil {
			fmt.Printf("Failed to create pod: %s\n", err)
		}
		return ctrl.Result{}, nil
	} else if err != nil {
		fmt.Printf("Reconcile Failed: %s\n", err)
		return ctrl.Result{}, nil
	}

	/* Start device plugin daemonSet if not already started */
	err = r.Get(context.TODO(),
		types.NamespacedName{
			Name: mctx.Spec.Acclr + "-sriovdp", Namespace: mctx.Namespace},
			driverSet)
	if errors.IsNotFound(err) {
		/* Start daemonSet */
		fmt.Printf("err %s\n", err)
		p := appsv1.DaemonSet{}
		byf, err := ioutil.ReadFile("/manifests/dev-plugin/" + 
				mctx.Spec.Acclr + "-sriovdp.yaml")
		if err != nil {
			fmt.Printf("Manifests not found: %s\n", err)
			return ctrl.Result{}, nil
		}
		yamlutil.Unmarshal(byf, &p)
		err = ctrl.SetControllerReference(mctx, &p, r.Scheme)
		if err != nil {
			fmt.Printf("Failed to set controller reference: %s\n", err)
			return ctrl.Result{}, nil
		}
		err = r.Create(ctx, &p)
		if err != nil {
			fmt.Printf("Failed to create pod: %s\n", err)
		}
		return ctrl.Result{}, nil
	} else if err != nil {
		fmt.Printf("Reconcile Failed: %s\n", err)
	}

	/* Start Driver validation Pods */
	driverPods := &corev1.PodList{}
	err = r.List(ctx, driverPods,
		client.MatchingLabels{"inline_mrvl_acclr_driver_ready": "false"})
	if err != nil {
		fmt.Printf("Error reading Podlist: %s\n", err)
		return ctrl.Result{}, nil
	}
	for _, s := range driverPods.Items {
		if s.Status.Phase == "Running" {
			valPods := &corev1.PodList{}
			r.List(ctx, valPods,
				client.MatchingLabels{"app": mctx.Spec.Acclr + "-drv-validate"},
				client.MatchingFields{"spec.nodeName": s.Spec.NodeName})
			if len(valPods.Items) == 0 {
				//fmt.Printf("170 driver Pod %s %s @ %s \n", s.Name, s.Status.Phase, s.Spec.NodeName)
				fmt.Printf("Start validator Pod\n")
				p := corev1.Pod{}
				byf, err := ioutil.ReadFile("/manifests/drv-daemon-validate/" + 
						mctx.Spec.Acclr + "-drv-validate.yaml")
				if err != nil {
					fmt.Printf("Manifests not found: %s\n", err)
					return ctrl.Result{}, nil
				}
				reg, _ := regexp.Compile(`nodeName: FILLED_BY_OPERATOR`)
				byf = reg.ReplaceAll(byf, []byte("nodeName: "+s.Spec.NodeName))
				reg, _ = regexp.Compile(`NAME_FILLED_BY_OPERATOR`)
				byf = reg.ReplaceAll(byf, []byte(s.Spec.NodeName))

				yamlutil.Unmarshal(byf, &p)
				err = ctrl.SetControllerReference(mctx, &p, r.Scheme)
				if err != nil {
					fmt.Printf("Failed to set controller reference: %s\n", err)
					return ctrl.Result{}, nil
				}
				err = r.Create(ctx, &p)
				if err != nil {
					fmt.Printf("Failed to create pod: %s\n", err)
				}
			}
		}
	}

	/* Check Driver Validation Pods */
	valPods := &corev1.PodList{}
	r.List(ctx, valPods, client.MatchingLabels{"app": "f95o-drv-validate"})
	for _, s := range valPods.Items {
		dPod := &corev1.PodList{}
		if s.Status.Phase == "Succeeded" {
			r.List(ctx, dPod,
				client.MatchingLabels{"inline_mrvl_acclr_driver_ready": "false"},
				client.MatchingFields{"spec.nodeName": s.Spec.NodeName})
			if len(dPod.Items) == 1 {
				dPod.Items[0].Labels["inline_mrvl_acclr_driver_ready"] = "true"
				err := r.Update(ctx, &dPod.Items[0])
				if err != nil {
					fmt.Printf("Failed to relabel %s: %s\n", s.Name, err)
				}
			}
			/* Update OctNicDevice and OctNicUpdaterStatus */
		} else if s.Status.Phase == "Failed" {
			// Delete driver pod
			r.List(ctx, dPod, client.MatchingLabels{"app": mctx.Spec.Acclr + "-driver"},
				client.MatchingFields{"spec.nodeName": s.Spec.NodeName})
			err = r.Delete(ctx, &dPod.Items[0])
			if err != nil {
				fmt.Printf("Failed to delete pod: %s\n", err)
			}
			// Delete the device plugin pod
			r.List(ctx, dPod,
				client.MatchingLabels{"app": mctx.Spec.Acclr + "-sriovdp"},
				client.MatchingFields{"spec.nodeName": s.Spec.NodeName})
			if len(dPod.Items) == 1 {
				err = r.Delete(ctx, &dPod.Items[0])
				if err != nil {
					fmt.Printf("Failed to delete pod: %s\n", err)
				}
			}

			// Delete the validation pod
			err = r.Delete(ctx, &s)
			if err != nil {
				fmt.Printf("Failed to delete pod: %s\n", err)
			}
		}
	}

	// Check device update pods
	updPods := &corev1.PodList{}
	r.List(ctx, updPods, client.MatchingLabels{"app": mctx.Spec.Acclr + "-dev-update"})
	for _, s := range updPods.Items {
		dPod := &corev1.PodList{}
		if s.Status.Phase == "Succeeded" {
			r.List(ctx, dPod,
					client.MatchingLabels{ "inline_mrvl_acclr_driver_ready": "Maintenance"},
					client.MatchingFields{"spec.nodeName": s.Spec.NodeName})
			if len(dPod.Items) == 1 {
				/* Restart driver pod */
				dPod.Items[0].Labels["inline_mrvl_acclr_driver_ready"] = "false"
				err := r.Update(ctx, &dPod.Items[0])
				if err != nil {
					fmt.Printf("Failed to Delete%s: %s\n", dPod.Items[0].Name, err)
				}
			}
			r.Delete(ctx, &s)
			if err != nil {
				fmt.Printf("Failed to delete pod %s: %s\n", s.Name, err)
			}
			mctx.Spec.Operation = "Run"
			err := r.Update(ctx, mctx)
			if err != nil {
					fmt.Printf("Failed to update mrvlAcclr: %s\n", err)
			}
		}
	}

	// Check CRD
	switch op := mctx.Spec.Operation; op {
	case "Maintenance":
		fallthrough
	case "maintenance":
		fallthrough
	case "MAINTENANCE":
		dPod := &corev1.PodList{}
		r.List(ctx, dPod,
			client.MatchingLabels{"app": mctx.Spec.Acclr + "-driver"},
			client.MatchingFields{"spec.nodeName": mctx.Spec.NodeName},
			)
		if len(dPod.Items) == 1 {
			mPod := &corev1.PodList{}
			r.List(ctx, mPod,
				client.MatchingLabels{"app": mctx.Spec.Acclr + "-driver"},
				client.MatchingLabels{"inline_mrvl_acclr_driver_ready": "Maintenance"},
				client.MatchingFields{"spec.nodeName": mctx.Spec.NodeName},
				)
			if len(mPod.Items) == 1 {
				// Already in maintenance
				return ctrl.Result{}, nil
			}
			// Not in Maintenance; Label driver POD in Maintenance.
			dPod.Items[0].Labels["inline_mrvl_acclr_driver_ready"] = "Maintenance"
			err := r.Update(ctx, &dPod.Items[0])
			if err != nil {
				fmt.Printf("Failed to relabel %s: %s\n",
				dPod.Items[0].Name, err)
			}
		}
		// Driver PoD does not exist OR is labeled Maintenance
		// Delete DP PoD
		r.List(ctx, dPod,
				client.MatchingLabels{"app": mctx.Spec.Acclr + "-sriovdp"},
				client.MatchingFields{"spec.nodeName": mctx.Spec.NodeName},
		)

		if len(dPod.Items) == 1 {
			err := r.Delete(ctx, &dPod.Items[0])
			if err != nil {
				fmt.Printf("Failed to Delete%s: %s\n", dPod.Items[0].Name, err)
			}
		}
		// Delete Driver Validation PoD
		r.List(ctx, dPod,
				client.MatchingLabels{"app": mctx.Spec.Acclr + "-drv-validation"},
				client.MatchingFields{"spec.nodeName": mctx.Spec.NodeName},
		)

		if len(dPod.Items) == 1 {
			err := r.Delete(ctx, &dPod.Items[0])
			if err != nil {
				fmt.Printf("Failed to Delete%s: %s\n", dPod.Items[0].Name, err)
			}
		}

		// Start Device Update PoD
		fmt.Printf("Start update Pod\n")
		p := corev1.Pod{}
		byf, err := ioutil.ReadFile("/manifests/dev-update/" +
						mctx.Spec.Acclr + "-update.yaml")
		if err != nil {
				fmt.Printf("Manifests not found: %s\n", err)
				return ctrl.Result{}, nil
		}

		fmt.Printf("NodeName: %s\n", mctx.Spec.NodeName);
		reg, _ := regexp.Compile(`nodeName: FILLED_BY_OPERATOR`)
		byf = reg.ReplaceAll(byf, []byte("nodeName: " + mctx.Spec.NodeName))
		reg, _ = regexp.Compile(`NAME_FILLED_BY_OPERATOR`)
		byf = reg.ReplaceAll(byf, []byte(mctx.Spec.NodeName))

		err = yamlutil.Unmarshal(byf, &p)
		if err != nil {
				fmt.Printf("%#v\n", p)
		}
		err = ctrl.SetControllerReference(mctx, &p, r.Scheme)
		if err != nil {
				fmt.Printf("Failed to set controller reference: %s\n", err)
				return ctrl.Result{}, nil
		}
		err = r.Create(ctx, &p)
		if err != nil {
				fmt.Printf("Failed to create update pod: %s\n", err)
				return ctrl.Result{}, nil
		}
	case "RUN":
		fallthrough
	case "Run":
		fallthrough
	case "run":
		fallthrough
	default:
		return ctrl.Result{}, nil
	}


	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OctNicUpdaterReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.Background(),
		&corev1.Pod{}, "spec.nodeName", func(rawObj client.Object) []string {
				pod := rawObj.(*corev1.Pod)
				return []string{pod.Spec.NodeName}
		}); err != nil {
			return err
	}

	c, err := controller.New("OctNicUpdater-controller", mgr,
			controller.Options{Reconciler: r, MaxConcurrentReconciles: 1})
	if err != nil {
		fmt.Printf("Failed to create new controller: %s\n", err)
		return err
	}
	err = c.Watch(&source.Kind{Type: &acclrv1beta1.OctNicUpdater{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		fmt.Printf("Failed to add watch controller: %s\n", err)
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:	&acclrv1beta1.OctNicUpdater{},
		})
	if err != nil {
		fmt.Printf("Failed to add watch controller: %s\n", err)
		return err
	}

	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:	&acclrv1beta1.OctNicUpdater{},
		})
	if err != nil {
		fmt.Printf("Failed to add watch controller: %s\n", err)
		return err
	}
	return nil
}
