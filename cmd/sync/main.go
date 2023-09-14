package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/yzxiu/calico-route-sync/pkg/calico"
	"github.com/yzxiu/calico-route-sync/pkg/controllers"
	"github.com/yzxiu/calico-route-sync/pkg/route"
	"github.com/yzxiu/calico-route-sync/pkg/util"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme         = runtime.NewScheme()
	setupLog       = ctrl.Log.WithName("setup")
	IpPoolResource = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "ippools",
	}
	NodeResource = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "nodes",
	}
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
}

func main() {

	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	stopCh := ctrl.SetupSignalHandler().Done()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		LeaderElection:     false,
		MetricsBindAddress: "0",
		Scheme:             scheme,
		Port:               9443,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	err = calico.AddToScheme(mgr.GetScheme())
	if err != nil {
		setupLog.Error(err, "unable to add blockaffinity scheme")
		os.Exit(1)
	}

	// Lister
	dynamicClient, err := dynamic.NewForConfig(mgr.GetConfig())
	if err != nil {
		panic(err.Error())
	}
	dynamicFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, 5*time.Minute)
	ippoolInformer := dynamicFactory.ForResource(IpPoolResource)
	nodeInformer := dynamicFactory.ForResource(NodeResource)
	dynamicFactory.Start(stopCh)
	for gvr, ok := range dynamicFactory.WaitForCacheSync(stopCh) {
		if !ok {
			panic(fmt.Sprintf("Failed to sync cache for resource %v", gvr))
		}
	}

	// localNetwork
	localNetworks, err := util.LocalNetworks()
	if err != nil {
		setupLog.Error(err, "unable to get local network")
		os.Exit(1)
	}
	router, err := route.NewRouter(localNetworks)

	r := &controllers.BlockAffinityReconciler{
		Client:       mgr.GetClient(),
		Log:          ctrl.Log.WithName("controllers").WithName("BlockAffinity"),
		Scheme:       mgr.GetScheme(),
		NodeLister:   nodeInformer.Lister(),
		IpPoolLister: ippoolInformer.Lister(),
		Router:       router,
	}
	if err = r.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "blockaffinity")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.TODO())
	go func() {
		<-stopCh
		setupLog.Info("clean pod route ...")
		r.CleanCalicoRoutes()
		setupLog.Info("clean pod route finished ...")
		cancel()
	}()

	setupLog.Info("starting manager ...")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager ...")
		os.Exit(1)
	}
}
