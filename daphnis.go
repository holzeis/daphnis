package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os/signal"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"os"

	"github.com/gorilla/mux"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("daphnis")
var clientset *kubernetes.Clientset

var templates = template.Must(template.ParseFiles("tmpl/services.html"))

type Service struct {
	Name  string
	Image string
	Chart string
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/info/{namespace}", Info).Methods("GET")
	server := &http.Server{Addr: ":8080", Handler: r}

	// initialize logger
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(logging.INFO, "")
	logging.SetBackend(backendLeveled)

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	go func() {
		log.Info("Listening on 0.0.0.0:8080")
		log.Fatal(server.ListenAndServe())
	}()

	// Setting up signal capturing
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Waiting for SIGINT (pkill -2)
	<-stop
	log.Warning("Stopping server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}

// Info fetches service information from the given namespace
func Info(w http.ResponseWriter, req *http.Request) {
	var vars map[string]string = mux.Vars(req)
	namespace := vars["namespace"]

	log.Infof("Searching for pods in %v", namespace)

	// list all pods in namespace
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		http.Error(w, fmt.Sprint(err.Error()), http.StatusInternalServerError)
		return
	}
	log.Infof("There are %d pods in the cluster\n", len(pods.Items))

	// fetch pod information
	services := []Service{}
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			service := Service{
				Chart: pod.GetObjectMeta().GetLabels()["chart"],
				Image: container.Image,
				Name:  container.Name,
			}
			services = append(services, service)
		}
	}

	// render service information to services template
	err = templates.ExecuteTemplate(w, "services.html", services)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
