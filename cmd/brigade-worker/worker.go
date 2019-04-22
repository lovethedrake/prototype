package main

import (
	"context"
	"log"

	"github.com/lovethedrake/prototype/pkg/brigade"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {

	clientConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	kubeClient, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		log.Fatal(err)
	}

	project, err := brigade.GetProjectFromEnvironmentAndSecret(kubeClient)
	if err != nil {
		log.Fatal(err)
	}

	event, err := brigade.GetEventFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	executor := brigade.NewExecutor(kubeClient)

	if err = executor.ExecuteBuild(
		context.Background(),
		project,
		event,
	); err != nil {
		log.Fatal(err)
	}

}
