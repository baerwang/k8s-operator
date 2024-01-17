package pkg

import (
	"context"
	"fmt"
	"path"
	"reflect"
	"time"

	v16 "k8s.io/api/core/v1"
	v12 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v15 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	v13 "k8s.io/client-go/informers/core/v1"
	v14 "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	vs "k8s.io/client-go/listers/core/v1"
	v1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	Client        kubernetes.Interface
	IngressLister v1.IngressLister
	ServiceLister vs.ServiceLister
	Queue         workqueue.RateLimitingInterface
}

func (c *Controller) addService(obj interface{}) {
	c.enqueue(obj)
}

func (c *Controller) updateService(obj interface{}, newObj interface{}) {
	if reflect.DeepEqual(obj, newObj) {
		return
	}
	c.enqueue(obj)
}

func (c *Controller) deleteIngress(obj interface{}) {
	ingress := obj.(*v12.Ingress)
	of := v15.GetControllerOf(ingress)
	if of == nil || of.Kind != "Service" {
		return
	}
	c.Queue.Add(path.Join(ingress.Namespace, ingress.Name))
}

func (c *Controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
	}
	c.Queue.Add(key)
}
func (c *Controller) Run(stop chan struct{}) {
	for i := 0; i < 5; i++ {
		go wait.Until(c.worker, time.Minute, stop)
	}
	<-stop
}

func (c *Controller) worker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	item, shutdown := c.Queue.Get()
	if shutdown {
		return false
	}
	c.Queue.Done(item)

	key := item.(string)
	if err := c.syncService(key); err != nil {
		c.handlerError(key, err)
	}
	return true
}

func (c *Controller) syncService(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	sl, err := c.ServiceLister.Services(namespace).Get(name)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	_, ok := sl.GetAnnotations()["ingress/http"]
	ingress, err := c.IngressLister.Ingresses(namespace).Get(name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if ok && errors.IsNotFound(err) {
		ig := c.constructIngress(sl)
		if _, err = c.Client.NetworkingV1().
			Ingresses(namespace).
			Create(context.Background(), ig, v15.CreateOptions{}); err != nil {
			return err
		}
	} else if !ok && ingress != nil {
		if err = c.Client.NetworkingV1().
			Ingresses(namespace).
			Delete(context.Background(), ingress.Name, v15.DeleteOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) handlerError(key string, err error) {
	fmt.Println("handlerError:", err)
	if c.Queue.NumRequeues(key) <= 10 {
		c.Queue.AddRateLimited(key)
		return
	}
	runtime.HandleError(err)
	c.Queue.Forget(err)
}

func (c *Controller) constructIngress(service *v16.Service) *v12.Ingress {
	ingress := v12.Ingress{}
	ingress.ObjectMeta.OwnerReferences = []v15.OwnerReference{*v15.NewControllerRef(service, v15.SchemeGroupVersion.WithKind("Service"))}

	ingress.Namespace = service.Namespace
	pathType := v12.PathTypePrefix
	ingress.Name = service.Name
	ig := "nginx"
	ingress.Spec = v12.IngressSpec{
		IngressClassName: &ig,
		Rules: []v12.IngressRule{{Host: "baerwang.com",
			IngressRuleValue: v12.IngressRuleValue{HTTP: &v12.HTTPIngressRuleValue{Paths: []v12.HTTPIngressPath{
				{Path: "/",
					PathType: &pathType,
					Backend: v12.IngressBackend{Service: &v12.IngressServiceBackend{Name: service.Name, Port: v12.ServiceBackendPort{
						Number: 80,
					}}}},
			}}},
		}}}

	return &ingress
}

func NewController(client kubernetes.Interface, serverInformer v13.ServiceInformer, ingressInformer v14.IngressInformer) Controller {
	c := Controller{
		Client:        client,
		ServiceLister: serverInformer.Lister(),
		IngressLister: ingressInformer.Lister(),
		Queue: workqueue.NewRateLimitingQueueWithConfig(
			workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
			workqueue.RateLimitingQueueConfig{
				Name: "ingressManager",
			}),
	}

	serverInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addService,
		UpdateFunc: c.updateService,
	})

	ingressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteIngress,
	})

	return c
}
