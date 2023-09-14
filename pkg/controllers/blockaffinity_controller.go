package controllers

import (
	"context"
	"encoding/json"
	"net"

	"github.com/go-logr/logr"
	"github.com/yzxiu/calico-route-sync/pkg/calico"
	"github.com/yzxiu/calico-route-sync/pkg/route"
	"github.com/yzxiu/calico-route-sync/pkg/util"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// BlockAffinityReconciler reconciles a BlockAffinity object
type BlockAffinityReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	NodeLister   cache.GenericLister
	IpPoolLister cache.GenericLister
	Router       *route.Router
}

func (r *BlockAffinityReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.Log.WithValues("BlockAffinity", req.NamespacedName)

	blockAffinity := &calico.BlockAffinity{}
	err := r.Get(ctx, req.NamespacedName, blockAffinity)

	if err != nil {
		r.Router.CheckRouters(r.getIpPoolsNets(), r.getBlockAffinityNets(ctx))
		if apierrs.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch blockaffinity")
		return ctrl.Result{}, err
	}

	log.Info("Reconciling BlockAffinity", "name", blockAffinity.Name)

	// delete
	if !blockAffinity.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info("Delete BlockAffinity", "name", blockAffinity.Name)
		err = r.Router.DeleteRoute(blockAffinity)
		if err != nil {
			r.Router.CheckRouters(r.getIpPoolsNets(), r.getBlockAffinityNets(ctx))
			log.Error(err, "del route error")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	node, err := r.getNode(blockAffinity.Spec.Node)
	if err != nil {
		r.Router.CheckRouters(r.getIpPoolsNets(), r.getBlockAffinityNets(ctx))
		if apierrs.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to get node")
		return ctrl.Result{}, err
	}
	nodeIP := util.NodeInternalIP(node)
	log.Info("Reconciling BlockAffinity", "node name", node.Name, "node ip", nodeIP)

	// update route
	err = r.Router.UpdateRoute(blockAffinity, nodeIP)
	if err != nil {
		log.Error(err, "update route error")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BlockAffinityReconciler) getNode(nodeName string) (*v1.Node, error) {
	uNode, err := r.NodeLister.Get(nodeName)
	if err != nil {
		return nil, err
	}
	bytes, err := runtime.Encode(unstructured.UnstructuredJSONScheme, uNode)
	var tNode v1.Node
	if err = json.Unmarshal(bytes, &tNode); err != nil {
		return nil, err
	}
	return &tNode, nil
}

func (r *BlockAffinityReconciler) getIpPoolsNets() []net.IPNet {
	var tPools []net.IPNet
	list, err := r.IpPoolLister.List(labels.Everything())
	if err != nil {
		return tPools
	}
	for _, pool := range list {
		bytes, err := runtime.Encode(unstructured.UnstructuredJSONScheme, pool)
		var tPool calico.IPPool
		if err = json.Unmarshal(bytes, &tPool); err != nil {
			continue
		}
		if !tPool.Spec.Disabled {
			n := util.ParseNet(tPool.Spec.CIDR)
			if n != nil {
				tPools = append(tPools, *n)
			}
		}
	}
	return tPools
}

func (r *BlockAffinityReconciler) getBlockAffinityNets(ctx context.Context) []net.IPNet {
	var bas []net.IPNet
	blockAffinityList := &calico.BlockAffinityList{}
	err := r.List(ctx, blockAffinityList)
	if err != nil {
		return bas
	}
	for _, ba := range blockAffinityList.Items {
		n := util.ParseNet(ba.Spec.CIDR)
		if n != nil {
			bas = append(bas, *n)
		}
	}
	return bas
}

func (r *BlockAffinityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&calico.BlockAffinity{}).
		Complete(r)
}

func (r *BlockAffinityReconciler) CleanCalicoRoutes() {
	r.Router.CleanRoutes(r.getIpPoolsNets())
}
