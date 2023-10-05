package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type KubeClient struct {
	*rest.Config
}

func NewKubeClient() (KubeClient, error) {
	conf, err := rest.InClusterConfig()
	if err != nil {
		return KubeClient{}, err
	}
	rest.SetKubernetesDefaults(conf)
	return KubeClient{
		conf,
	}, nil
}

// CreateObject creates an object in a namespace. Authentication and api client settings are taken from the instancer config.
// This procedure first requests the apiserver for the mapping of the object Kind to object Resource, then makes the request
// to create an object of that Resource using unstructObj as the specification.
// https://book.kubebuilder.io/cronjob-tutorial/gvks.html
func (k *KubeClient) CreateObject(unstructObj *unstructured.Unstructured, namespace string) (*unstructured.Unstructured, error) {
	resource, err := k.GetObjectResource(unstructObj)
	if err != nil {
		return nil, err
	}

	client, err := dynamic.NewForConfig(k.Config)
	if err != nil {
		return nil, err
	}

	resObj, err := client.Resource(resource).Namespace(namespace).Create(context.TODO(), unstructObj, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return resObj, nil
}

func (k *KubeClient) DeleteObject(unstructObj *unstructured.Unstructured, namespace string) error {
	resource, err := k.GetObjectResource(unstructObj)
	if err != nil {
		return err
	}

	deletePolicy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}

	client, err := dynamic.NewForConfig(k.Config)
	if err != nil {
		return err
	}

	if err := client.Resource(resource).Namespace(namespace).Delete(context.TODO(), unstructObj.GetName(), deleteOptions); err != nil {
		return err
	}

	return nil
}

func (k *KubeClient) ListObjects(conf *rest.Config, unstructObj *unstructured.Unstructured, namespace string) ([]string, error) {
	resource, err := k.GetObjectResource(unstructObj)
	if err != nil {
		return nil, err
	}

	client, err := dynamic.NewForConfig(k.Config)
	if err != nil {
		return nil, err
	}

	list, err := client.Resource(resource).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// todo: replace example code below
	for _, d := range list.Items {
		replicas, found, err := unstructured.NestedInt64(d.Object, "spec", "replicas")
		if err != nil || !found {
			fmt.Printf("Replicas not found for deployment %v: error=%v", d.GetName(), err)
			continue
		}
		fmt.Printf(" * %v (%d replicas)\n", d.GetName(), replicas)
	}
	return nil, nil
}

func (k *KubeClient) GetObjectResource(unstructObj *unstructured.Unstructured) (schema.GroupVersionResource, error) {
	c, err := discovery.NewDiscoveryClientForConfig(k.Config)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	groupResources, err := restmapper.GetAPIGroupResources(c)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	mapping, err := mapper.RESTMapping(unstructObj.GetObjectKind().GroupVersionKind().GroupKind())
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	return mapping.Resource, nil
}
