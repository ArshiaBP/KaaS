package configs

import (
	"flag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"path/filepath"
)

var Client *kubernetes.Clientset

func CreateClient() {
	var kubeConfig *string
	home := homedir.HomeDir()
	kubeConfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "")
	flag.Parse()
	//access from outside the cluster
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
	if err != nil {
		log.Fatal(err)
	}
	//access from inside the cluster
	//config, err := rest.InClusterConfig()
	//if err != nil {
	//	log.Fatal(err)
	//}
	Client, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
}
