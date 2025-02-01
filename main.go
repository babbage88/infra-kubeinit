package main

import (
	"fmt"
	"time"

	"github.com/babbage88/infra-kubeinit/internal/pretty"
	batchv1 "k8s.io/api/batch/v1"
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

func (k *KubeClient) PrepDeployment() error {
	// Retrieve all migration jobs
	jobsList, err := k.GetBatchJobByLabel("default", "workload-type=db-migration")
	if err != nil {
		pretty.PrintErrorf("Encountered Error: %s", err.Error())
		return fmt.Errorf("Error retrieving batch Jobs %w", err)
	}

	// Find the latest successful job
	latestJob := getLatestSuccessfulJob(jobsList.Items)
	if latestJob == nil {
		pretty.PrintWarning("No successful migration jobs found.")
		pretty.Print("Creating Migration Job")
		fmt.Println()
		ttl := int32(120)
		err := k.CreateBatchJob("init-db", "default", "ghcr.io/babbage88/init-infradb:v1.0.9", "initdb-env", "initdb.env", &ttl)
		if err != nil {
			return fmt.Errorf("Error creating DB migration Job %w", err)
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
			err := k.CreateBatchJob("init-db", "default", "ghcr.io/babbage88/init-infradb:v1.0.9", "initdb-env", "initdb.env", &ttl)
			if err != nil {
				return fmt.Errorf("Error creating DB migration Job %w", err)
			}
			return err

		} else {
			pretty.Print("Last successful job is recent. No need to create a new job.")
			return err
		}
	} else {
		pretty.PrintWarning("Job status found, but CompletionTime is nil. Creating a new job.")
		ttl := int32(120)
		err := k.CreateBatchJob("init-db", "default", "ghcr.io/babbage88/init-infradb:v1.0.9", "initdb-env", "initdb.env", &ttl)
		if err != nil {
			return fmt.Errorf("Error creating DB migration Job %w", err)
		}
		return err
	}
}

func main() {
	// Initialize Kubernetes client
	kubeClient := NewKubeClient()
	kubeClient.InitializeExternalClient()

	//// Debug output of job statuses
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
