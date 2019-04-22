package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	gin "gopkg.in/gin-gonic/gin.v1"

	v1 "k8s.io/api/core/v1"

	"github.com/brigadecore/brigade/pkg/storage"
	"github.com/brigadecore/brigade/pkg/storage/kube"
	"github.com/brigadecore/brigade/pkg/webhook"
)

var (
	kubeconfig string
	master     string
	namespace  string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&master, "master", "", "master url")
	flag.StringVar(&namespace, "namespace", defaultNamespace(), "kubernetes namespace")
}

func main() {
	flag.Parse()

	clientset, err := kube.GetClient(master, kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	if namespace == "" {
		namespace = v1.NamespaceDefault
	}

	store := kube.New(clientset, namespace)

	router := newRouter(store)
	router.Run(":8000")
}

func newRouter(store storage.Store) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	handlers := map[string]gin.HandlerFunc{
		"/simpleevents/v1": webhook.NewGenericWebhookSimpleEvent(store),
		"/cloudevents/v02": webhook.NewGenericWebhookCloudEvent(store),
	}

	for endpoint, handler := range handlers {
		events := router.Group(endpoint)
		events.Use(gin.Logger())
		events.POST("/:projectID/:secret", handler)
	}

	router.GET("/healthz", healthz)
	return router
}

func healthz(c *gin.Context) {
	c.String(http.StatusOK, http.StatusText(http.StatusOK))
}

func defaultNamespace() string {
	if ns, ok := os.LookupEnv("BRIGADE_NAMESPACE"); ok {
		return ns
	}
	return v1.NamespaceDefault
}
