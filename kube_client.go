package main

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/babbage88/infra-kubeinit/internal/pretty"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1util "k8s.io/apimachinery/pkg/util/intstr"
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

// createJob creates a Kubernetes Job using client-go
func (k *KubeClient) CreateBatchJob(jobName string, namespace string, imageName string, volName string, secretName string, ttl *int32) error {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: jobName,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: ttl,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"workload":      "job",
						"app":           "go-infra",
						"workload-type": "db-migration",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:            jobName,
							Image:           imageName,
							ImagePullPolicy: corev1.PullAlways,
							Command:         []string{"/app/migrate"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volName,
									MountPath: "/app/.env",
									SubPath:   ".env",
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("512Mi"),
									corev1.ResourceCPU:    resource.MustParse("500m"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
									corev1.ResourceCPU:    resource.MustParse("250m"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: volName,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: secretName,
								},
							},
						},
					},
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: "ghcr"},
					},
				},
			},
		},
	}

	// Create the Job
	jobsClient := k.Client.BatchV1().Jobs(namespace)
	_, err := jobsClient.Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		slog.Error("failed to create job", slog.String("error", err.Error()))
		return fmt.Errorf("Error creating job %w", err)
	}

	pretty.Printf("Job created successfully %s", job.Name)
	slog.Info("Job created successfully\n", slog.String("name", job.Name))
	return nil
}

func (k *KubeClient) CreateDeployment(namespace *string, deploymentName *string, replicas *int32, imageName *string, containerPort *int32) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: *deploymentName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": *deploymentName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": *deploymentName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            *deploymentName,
							Image:           *imageName,
							ImagePullPolicy: corev1.PullAlways,
							Command:         []string{"/app/server"},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: *containerPort,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "cf-token-ini",
									MountPath: "/run/secrets/cf_token.ini",
									SubPath:   "cf_token.ini",
								},
								{
									Name:      "k3s-env",
									MountPath: "/app/.env",
									SubPath:   "k3s.env",
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("512Mi"),
									corev1.ResourceCPU:    resource.MustParse("500m"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
									corev1.ResourceCPU:    resource.MustParse("250m"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "cf-token-ini",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "cf-token-ini",
								},
							},
						},
						{
							Name: "k3s-env",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "k3s-env",
								},
							},
						},
					},
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: "ghcr"},
					},
				},
			},
		},
	}

	// Apply Deployment
	deploymentsClient := k.Client.AppsV1().Deployments(*namespace)
	_, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		slog.Error("Error creating deployment", slog.String("error", err.Error()))
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	slog.Info("Deployment created successfully", slog.String("deploymentName", *deploymentName))
	return nil
}

func (k *KubeClient) CreateLoadBalancerService(namespace *string, serviceName *string, targetPort *int32, exposedPort *int32, appLabel *string, allocateNodePort *bool) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *serviceName,
			Namespace: *namespace,
			Labels: map[string]string{
				"app": *appLabel,
			},
		},
		Spec: corev1.ServiceSpec{
			AllocateLoadBalancerNodePorts: allocateNodePort,
			Selector: map[string]string{
				"app": *appLabel,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       *exposedPort, // Exposed service port
					TargetPort: metav1util.IntOrString{Type: metav1util.Int, IntVal: *targetPort},
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeLoadBalancer, // Exposes the service externally
		},
	}

	// Create the Service
	servicesClient := k.Client.CoreV1().Services(*namespace)
	_, err := servicesClient.Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create LoadBalancer Service: %w", err)
	}

	slog.Info("LoadBalancer Service go-infra-service created successfully")
	return nil
}
