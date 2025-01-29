package main

import (
	"encoding/json"
	"fmt"

	"github.com/babbage88/infra-kubeinit/internal/pretty"
)

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
		pretty.PrintError("No successful migration jobs found.")
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
