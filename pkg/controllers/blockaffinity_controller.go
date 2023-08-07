package controllers

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/yzxiu/calico-route-sync/pkg/calico"
	"github.com/yzxiu/calico-route-sync/pkg/route"
	"github.com/yzxiu/calico-route-sync/pkg/util"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	v1 "k8s.io/client-go/listers/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// BlockAffinityReconciler reconciles a BlockAffinity object
type BlockAffinityReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	NodeLister v1.NodeLister
	Router     *route.Router
}

func (r *BlockAffinityReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.Log.WithValues("BlockAffinity", req.NamespacedName)

	ba := &calico.BlockAffinity{}
	err := r.Get(ctx, req.NamespacedName, ba)

	if err != nil {
		if apierrs.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch blockaffinity")
		return ctrl.Result{}, err
	}

	log.Info("Reconciling BlockAffinity", "name", ba.Name)

	node, err := r.NodeLister.Get(ba.Spec.Node)
	if err != nil {
		if apierrs.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to get node")
		return ctrl.Result{}, err
	}
	nodeIP := util.NodeInternalIP(node)
	log.Info("Reconciling BlockAffinity", "node name", node.Name, "node ip", nodeIP)

	// delete
	if !ba.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info("Delete BlockAffinity", "name", ba.Name)
		err = r.Router.DeleteRoute(ba, nodeIP)
		if err != nil {
			log.Error(err, "del route error")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// update route
	err = r.Router.UpdateRoute(ba, nodeIP)
	if err != nil {
		log.Error(err, "update route error")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BlockAffinityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&calico.BlockAffinity{}).
		Complete(r)
}
