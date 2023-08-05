package main

import (
	"context"
	"flag"
	"github.com/yzxiu/calico-route-sync/pkg/calico"
	"github.com/yzxiu/calico-route-sync/pkg/controllers"
	"github.com/yzxiu/calico-route-sync/pkg/route"
	"github.com/yzxiu/calico-route-sync/pkg/util"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
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

	// NodeLister
	client, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		panic(err.Error())
	}
	factory := informers.NewSharedInformerFactory(client, 5*time.Minute)
	nodeInformer := factory.Core().V1().Nodes()
	go factory.Start(stopCh)
	// Waiting for caches to sync for node
	cache.WaitForNamedCacheSync("node", stopCh, nodeInformer.Informer().HasSynced)

	// localNetwork
	localNetworks, err := util.LocalNetworks()
	if err != nil {
		setupLog.Error(err, "unable to get local network")
		os.Exit(1)
	}
	router, err := route.NewRouter(localNetworks)

	r := &controllers.BlockAffinityReconciler{
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("controllers").WithName("BlockAffinity"),
		Scheme:     mgr.GetScheme(),
		NodeLister: nodeInformer.Lister(),
		Router:     router,
	}
	if err = r.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "blockaffinity")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.TODO())
	go func() {
		<-stopCh
		setupLog.Info("clean pod route ...")
		list := calico.NewBlockAffinityList()
		err = r.List(ctx, list)
		for _, af := range list.Items {
			er := route.CleanupLeftovers(af.Spec.CIDR)
			if !er {
				setupLog.Info("del route", "route", af.Spec.CIDR)
			}
		}
		setupLog.Info("clean pod route finished ...")
		cancel()
	}()

	setupLog.Info("starting manager ...")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager ...")
		os.Exit(1)
	}
}
