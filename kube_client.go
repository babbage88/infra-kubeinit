package main

import (
	"log/slog"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type KubeClient struct {
	Client         *kubernetes.Clientset `json:"client"`
	KubeconfigPath string                `json:"kubeconfigPath"`
}

// Initializes client from pod running inside of a cluster.
func (k *KubeClient) InitializeInternalClient() error {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		slog.Error("Error Initializing Internal KubeClient", slog.String("error", err.Error()))
		return err
	}
	// creates the clientset
	k.Client, err = kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("Error Initializing Clientset for KubeClient", slog.String("error", err.Error()))
		return err

	}

	return err
}

// Initialize client from outside of a cluster.
func (k *KubeClient) InitializeExternalClient() error {
	home := homedir.HomeDir()
	if k.KubeconfigPath == "" {
		k.KubeconfigPath = filepath.Join(home, ".kube", "config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", k.KubeconfigPath)
	if err != nil {
		slog.Error("Error Initializing Internal KubeClient", slog.String("error", err.Error()))
		return err
	}

	// creates the clientset
	k.Client, err = kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("Error Initializing Clientset for KubeClient", slog.String("error", err.Error()))
		return err

	}

	return err
}

func NewDefaultExternalKubeClient() (*KubeClient, error) {
	k := &KubeClient{}
	err := k.InitializeExternalClient()
	if err != nil {
		slog.Error("Error Initializing External Clientset for KubeClient", slog.String("error", err.Error()))
		return k, err
	}

	return k, err
}

func NewInternalKubeClient() (*KubeClient, error) {
	k := &KubeClient{}
	err := k.InitializeInternalClient()
	if err != nil {
		slog.Error("Error Initializing External Clientset for KubeClient", slog.String("error", err.Error()))
		return k, err
	}

	return k, err
}

func (k *KubeClient) GetPods(namespace string) v1.PodInterface {
	podInt := k.Client.CoreV1().Pods(namespace)
	return podInt
}
