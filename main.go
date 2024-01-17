package main

import (
	"k8s-operator/pkg"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		clusterConfig, err := rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
		config = clusterConfig
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	factory := informers.NewSharedInformerFactory(clientSet, 0)
	services := factory.Core().V1().Services()
	ingresses := factory.Networking().V1().Ingresses()

	c := pkg.NewController(clientSet, services, ingresses)
	stop := make(chan struct{})
	factory.Start(stop)
	factory.WaitForCacheSync(stop)
	c.Run(stop)
}
