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
	obj := unstructured.Unstructured{}
	err := yaml.UnmarshalStrict([]byte(manifest), &obj)
	if err != nil {
		return nil, err
	}
	return &obj, nil
}

// UnmarshalManifestFile umarshals a yaml string with multiple objects
// delimited by '---' and returns a list of objects in it
func UnmarshalManifestFile(content string) ([]unstructured.Unstructured, error) {
	objYamls := strings.Split(content, "---")
	res := make([]unstructured.Unstructured, len(objYamls))
	for _, v := range objYamls {
		if v == "\n" || v == "" {
			// ignore empty cases
			continue
		}
		obj, err := UnmarshalSingleManifest(v)
		if err != nil {
			return nil, err
		}
		res = append(res, *obj)
	}
	return res, nil
}

func (in *Instancer) UnmarshalChallenges(challenges map[string]string) (map[string][]unstructured.Unstructured, error) {
	res := make(map[string][]unstructured.Unstructured, len(challenges))
	for k, v := range challenges {
		objs, err := UnmarshalManifestFile(v)
		if err != nil {
			return nil, err
		}
		res[k] = objs
	}
	return res, nil
}

// CreateObject creates an object in a namespace. Authentication and api client settings are taken from the instancer config.
// This procedure first requests the apiserver for the mapping of the object Kind to object Resource, then makes the request
// to create an object of that Resource using unstructObj as the specification.
// https://book.kubebuilder.io/cronjob-tutorial/gvks.html
func (in *Instancer) CreateObject(unstructObj *unstructured.Unstructured, namespace string) (*unstructured.Unstructured, error) {

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

	resObj, err := client.Resource(mapping.Resource).Namespace(namespace).Create(context.TODO(), unstructObj, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return resObj, nil
}
