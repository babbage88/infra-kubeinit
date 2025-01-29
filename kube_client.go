package main

import (
	"context"
	"log"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/babbage88/infra-kubeinit/internal/pretty"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type IKubeClient interface {
	New(opts ...KubeClientOption) *KubeClient
}

type KubeClient struct {
	Client         *kubernetes.Clientset `json:"client"`
	KubeconfigPath string                `json:"kubeconfigPath"`
	Ctx            context.Context       `json:"context"`
}

type KubeClientOption func(k *KubeClient)

func WithKubeconfigPath(s string) KubeClientOption {
	return func(k *KubeClient) {
		k.KubeconfigPath = s
	}
}

func WithContext(ctx context.Context) KubeClientOption {
	return func(k *KubeClient) {
		k.Ctx = ctx
	}
}

func NewKubeClient(opts ...KubeClientOption) *KubeClient {
	home := homedir.HomeDir()
	defaultKubeconfigPath := filepath.Join(home, ".kube", "config")
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(time.Second*2))
	k := &KubeClient{
		KubeconfigPath: defaultKubeconfigPath,
		Ctx:            ctx,
	}

	for _, opt := range opts {
		opt(k)
	}

	return k
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

func (k *KubeClient) GetPods(namespace string, podName string) (*corev1.Pod, error) {
	pod, err := k.Client.CoreV1().Pods(namespace).Get(k.Ctx, podName, metav1.GetOptions{})
	if err != nil {
		slog.Error("Error getting pods", slog.String("error", err.Error()))
	}
	pretty.Printf("Pod %s found in %s\n", podName, namespace)
	pretty.Printf("Name: %s\n", pod.GetName())
	pretty.Printf("uid: %s\n", pod.UID)
	pretty.Printf("Created: %s\n", pod.CreationTimestamp.String())

	return pod, err
}

func (k *KubeClient) ListJobs(namespace string) (*batchv1.JobList, error) {
	job, err := k.Client.BatchV1().Jobs(namespace).List(k.Ctx, metav1.ListOptions{})
	if err != nil {
		slog.Error("Error getting pods", slog.String("error", err.Error()))
	}

	return job, err
}

func (k *KubeClient) GetBatchJobByLabel(namespace string, label string) (*batchv1.JobList, error) {
	job, err := k.Client.BatchV1().Jobs(namespace).List(k.Ctx, metav1.ListOptions{LabelSelector: label})
	if err != nil {
		slog.Error("Error retrieving jobs", slog.String("error", err.Error()))
	}

	return job, err
}

func (k *KubeClient) LaunchBatchJob(namespace *string, jobName *string, image *string, cmd *string) {
	jobs := k.Client.BatchV1().Jobs(*namespace)
	var backOffLimit int32 = 0

	jobSpec := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *jobName,
			Namespace: *namespace,
		},
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:    *jobName,
							Image:   *image,
							Command: strings.Split(*cmd, " "),
						},
					},
					RestartPolicy: v1.RestartPolicyNever,
				},
			},
			BackoffLimit: &backOffLimit,
		},
	}

	_, err := jobs.Create(context.TODO(), jobSpec, metav1.CreateOptions{})
	if err != nil {
		log.Fatalln("Failed to create K8s job.")
	}

	// print job details
	slog.Info("Job created successfully")
}
