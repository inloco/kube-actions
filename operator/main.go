package main

import (
	"context"
	"fmt"
	"log"

	"github.com/inloco/kube-actions/operator/controller"

	inlocov1alpha1 "github.com/inloco/kube-actions/operator/pkg/apis/inloco/v1alpha1"
	inlocov1alpha1client "github.com/inloco/kube-actions/operator/pkg/generated/clientset/versioned/typed/inloco/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	//kubeconfig, err := restclient.InClusterConfig()
	//if err != nil {
	//	log.Panic(err)
	//}

	ctx := context.Background()

	kubeconfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		log.Panic(err)
	}

	client, err := inlocov1alpha1client.NewForConfig(kubeconfig)
	if err != nil {
		log.Panic(err)
	}

	events, err := client.ActionsRunners("").Watch(ctx, metav1.ListOptions{})
	if err != nil {
		log.Panic(err)
	}

	controllers := make(map[string]*controller.RunnerController)
	for event := range events.ResultChan() {
		actionsRunner := event.Object.(*inlocov1alpha1.ActionsRunner)
		key := fmt.Sprintf("%s/%s", actionsRunner.GetNamespace(), actionsRunner.GetName())

		switch typ := event.Type; typ {
		case watch.Added, watch.Modified:
			if ctrl, ok := controllers[key]; ok {
				if err := ctrl.Notify(); err != nil {
					log.Printf("failed %#v for %#v: %v", event, actionsRunner, err)
					continue
				}
			} else {
				ctrl, err := controller.NewRunnerController(kubeconfig, actionsRunner)
				if err != nil {
					log.Printf("failed %#v for %#v: %v", event, actionsRunner, err)
					continue
				}
				controllers[key] = ctrl
				go func() {
					if err := ctrl.Loop(); err != nil {
						log.Panic(err)
					}
				}()
			}

		case watch.Deleted:
			if ctrl, ok := controllers[key]; ok {
				if err := ctrl.Close(); err != nil {
					log.Printf("failed %#v for %#v: %v", event, actionsRunner, err)
					continue
				}
				delete(controllers, key)
			}

		default:
			log.Printf("ingored %#v for %#v", typ, actionsRunner)
			continue
		}

		log.Printf("consumed %#v for %#v", event, actionsRunner)
	}
}
