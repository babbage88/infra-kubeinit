package main

import (
	"encoding/json"
	"fmt"

	"github.com/babbage88/infra-kubeinit/internal/pretty"
)

func int32Ptr(i int32) *int32 {
	return &i
}

func main() {
	pretty.Print("Starting main")
	kubeClient := NewKubeClient()
	kubeClient.InitializeExternalClient()

	jobsList, err := kubeClient.GetBatchJobByLabel("default", "workload-type=db-migration")
	if err != nil {
		pretty.PrintErrorf("Encountered Error: %s", err.Error())
	}
	jobListLength := len(jobsList.Items)
	if jobListLength < 1 {
		pretty.PrintWarning("No successful migration jobs found.")
		pretty.Print("Creating Migration Job")
		ttl := int32(120)
		kubeClient.CreateBatchJob("init-db", "default", "ghcr.io/babbage88/init-infradb:v1.0.9", "initdb-env", "initdb.env", int32Ptr(ttl))
	}

	fmt.Printf("Length of JobList: %d\n", jobListLength)
	for _, j := range jobsList.Items {
		response, err := json.MarshalIndent(j.Status, "", "  ")
		if err != nil {
			pretty.PrintErrorf("Errror Marsharling pretty reponse")
		}

		msg := fmt.Sprint(string(response))
		pretty.Print(msg)
		fmt.Println()
	}
}
