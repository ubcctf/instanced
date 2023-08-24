package main

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

// UnmarshalSingleManifest unmarshals a single object in yaml string form.
// Objects after the first separated by '---' are ignored.
func UnmarshalSingleManifest(manifest string) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	err := yaml.UnmarshalStrict([]byte(manifest), obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (in *Instancer) CreateObject(manifest string, namespace string) (*unstructured.Unstructured, error) {
	unstructObj, err := UnmarshalSingleManifest(manifest)
	if err != nil {
		return nil, err
	}
	in.echo.Logger.Debug(unstructObj)

	c, err := discovery.NewDiscoveryClientForConfig(in.k8sConfig)
	if err != nil {
		return nil, err
	}

	groupResources, err := restmapper.GetAPIGroupResources(c)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	mapping, err := mapper.RESTMapping(unstructObj.GetObjectKind().GroupVersionKind().GroupKind())
	if err != nil {
		return nil, err
	}
	client, err := dynamic.NewForConfig(in.k8sConfig)
	if err != nil {
		return nil, err
	}
	resObj, err := client.Resource(mapping.Resource).Namespace(namespace).Create(context.Background(), unstructObj, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return resObj, nil
}

func SplitYaml(content string) []string {
	return strings.Split(content, "---")
}
