package main

import (
	"context"
	"fmt"
	"log/slog"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// Get the generic dynamic Resource Client
func getGenericResourceClient(resourceType string, group string, version string) dynamic.ResourceInterface {
	genericSchema := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resourceType,
	}

	dynamicClient, err := dynamic.NewForConfig(&rest.Config{})
	if err != nil {
		slog.Error("error creating dynamic client", slog.String("error", err.Error()))
	}
	// Attach the given resource and schema
	return dynamicClient.Resource(genericSchema).Namespace("Namespace")
}

// Main method to deploy any type of kubernets kind
func genericDeployment(resourceType string, resourceName string, resourceObject runtime.Object, group string, version string) error {
	// Context to consider
	Ctx := context.Background()

	// 1. Get the dynamic resouce client
	resourceClient := getGenericResourceClient(resourceType, group, version)

	// 2. convert runtimeObject to unstructured
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(resourceObject)
	if err != nil {
		return fmt.Errorf("Found error while coverting resource to unstructured err - %s", err)
	}
	unstructuredResource := &unstructured.Unstructured{Object: unstructuredObj}

	// 3. try to see if the resource exists
	existingResource, err := resourceClient.Get(context.Background(), resourceName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Resource doesn't exist, so create one
			_, err = resourceClient.Create(Ctx, unstructuredResource, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("Found error while creating the Resource : %s, err - %s", resourceName, err)
			}
		} else {
			return fmt.Errorf("Found error initialising the client : %s, err - %s", resourceType, err)
		}
	} else {
		// Resource already exists, so update the existing resource
		existingResource.Object = unstructuredObj
		_, err := resourceClient.Update(Ctx, existingResource, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("Found error while Updating %s, - err : %s", resourceType, err)
		}
	}

	return nil
}
