package main

import (
	"flag"
	"fmt"
	"log/slog"
	"time"

	"github.com/babbage88/infra-kubeinit/internal/bumper"
	"github.com/babbage88/infra-kubeinit/internal/pretty"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/util/homedir"
)

var (
	kubeConfigPath string
	home           string = homedir.HomeDir()
)

func getLatestSuccessfulJob(jobsList []batchv1.Job) *batchv1.Job {
	var latestJob *batchv1.Job
	var latestCompletionTime time.Time

	for _, job := range jobsList {
		for _, condition := range job.Status.Conditions {
			if condition.Type == batchv1.JobComplete && condition.Status == "True" {
				if condition.LastTransitionTime.Time.After(latestCompletionTime) {
					latestCompletionTime = condition.LastTransitionTime.Time
					latestJob = &job
				}
			}
		}
	}
	return latestJob
}

func (k *KubeClient) PrepDeployment(initDbImage string) error {
	// Retrieve all migration jobs
	jobsList, err := k.GetBatchJobByLabel("default", "workload-type=db-migration")
	pretty.PrettyPrintK8sJob(jobsList)
	if err != nil {
		pretty.PrintErrorf("Encountered Error: %s", err.Error())
		return fmt.Errorf("error retrieving batch jobs %w", err)
	}

	// Find the latest successful job
	latestJob := getLatestSuccessfulJob(jobsList.Items)
	if latestJob == nil {
		pretty.PrintWarning("No successful migration jobs found.")
		pretty.Print("Creating Migration Job")
		fmt.Println()
		ttl := int32(120)
		err := k.CreateBatchJob("init-db", "default", initDbImage, "initdb-env", "initdb.env", &ttl)
		if err != nil {
			return fmt.Errorf("error creating database migration Job %w", err)
		}
		return err
	}

	// Check if the latest successful job was completed more than 2 minutes ago
	latestCompletionTime := latestJob.Status.CompletionTime
	if latestCompletionTime != nil {
		timeSinceCompletion := time.Since(latestCompletionTime.Time)
		if timeSinceCompletion > 2*time.Minute {
			pretty.Print("Last successful job completed more than 2 minutes ago. Creating a new job.")
			ttl := int32(120)
			err := k.CreateBatchJob("init-db", "default", initDbImage, "initdb-env", "initdb.env", &ttl)
			if err != nil {
				return fmt.Errorf("error creating database migration job %w", err)
			}
			return err

		} else {
			pretty.Print("Last successful job is recent. No need to create a new job.")
			return err
		}
	} else {
		pretty.PrintWarning("Job status found, but CompletionTime is nil. Creating a new job.")
		ttl := int32(120)
		err := k.CreateBatchJob("init-db", "default", initDbImage, "initdb-env", "initdb.env", &ttl)
		if err != nil {
			return fmt.Errorf("error creating database migration job %w", err)
		}
		return err
	}
}

type Cast interface {
	IntToInt32(i *int) *int32
}

func IntToInt32(i *int) *int32 {
	i32 := int32(*i)
	return &i32
}

func main() {
	flag.StringVar(&kubeConfigPath, "kubeconfig", fmt.Sprintf("%s/.kube/config", home), "kubeconfig file to use")
	containerPort := flag.Int("container-port", 8993, "Container port")
	runBumper := flag.Bool("bumper", false, "Used to calculate next release version number")
	bumpType := flag.String("increment-type", "patch", "major, minor, patch")
	currentVersion := flag.String("latest-version", "", "Version number to increment eg: v1.2.2")
	namespace := flag.String("namespace", "default", "Namespace for deployment")
	deploymentName := flag.String("deployment-name", "go-infra", "deploymenyt name")
	serviceName := flag.String("service-name", "go-infra-svc", "Service Name")
	replicas := flag.Int("replicas", 3, "Number of replicas in deployment")
	dbMigrationImageName := flag.String("dbinit-image-name", "ghcr.io/babbage88/init-infradb:v1.2.2", "Image name to user for DB Migration init")
	imageName := flag.String("image-name", "ghcr.io/babbage88/go-infra:v1.2.2", "Image name to user for deployment")
	allocateNodePort := flag.Bool("allocate-nodeport", false, "Allocate NodePort for LoadBalancer deployment")
	deployService := flag.Bool("deploy-service", false, "Deploy LoadBalancer service")
	flag.Parse()

	if *runBumper {
		bumper.BumpVersion(*currentVersion, *bumpType)
		return
	}

	// Initialize Kubernetes client
	kubeClient := NewKubeClient(WithKubeconfigPath(kubeConfigPath))
	kubeClient.InitializeExternalClient()
	err := kubeClient.PrepDeployment(*dbMigrationImageName)
	if err != nil {
		pretty.PrintErrorf("Error prepping deployment error: %s", err.Error())
		slog.Error("Error prepping deployment", slog.String("error", err.Error()))
	}

	if *deployService {
		pretty.Print("Creating or Updating deployment...")
		err = kubeClient.CreateOrUpdateDeployment(namespace, deploymentName, IntToInt32(replicas), imageName, IntToInt32(containerPort))
		if err != nil {
			slog.Error("Error Creating Deplyment", slog.String("error", err.Error()))
		}
		pretty.Print("deployment created")

		err = kubeClient.CreateLoadBalancerService(namespace, serviceName, IntToInt32(containerPort), IntToInt32(containerPort), deploymentName, allocateNodePort)
		if err != nil {
			slog.Error("error creating service", slog.String("error", err.Error()))
		}
		pretty.Print("Service created")

	}

	//// Debug output of job statusesc
	//for _, j := range jobsList.Items {
	//	fmt.Println()
	//	response, err := json.MarshalIndent(j.Status, "", "  ")
	//	if err != nil {
	//		pretty.PrintErrorf("Error marshaling response: %s", err.Error())
	//		continue
	//	}
	//	pretty.Print(string(response))
	//	fmt.Println()
	//}
}
