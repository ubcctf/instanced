package main

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	res := make([]unstructured.Unstructured, 0)
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

func UnmarshalChallenges(challenges map[string]string) (map[string][]unstructured.Unstructured, error) {
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
	resource, err := in.GetObjectResource(unstructObj)
	if err != nil {
		return nil, err
	}

	client, err := dynamic.NewForConfig(in.k8sConfig)
	if err != nil {
		return nil, err
	}

	resObj, err := client.Resource(resource).Namespace(namespace).Create(context.TODO(), unstructObj, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return resObj, nil
}

func (in *Instancer) DeleteObject(unstructObj *unstructured.Unstructured, namespace string) error {
	resource, err := in.GetObjectResource(unstructObj)
	if err != nil {
		return err
	}

	deletePolicy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}

	client, err := dynamic.NewForConfig(in.k8sConfig)
	if err != nil {
		return err
	}

	if err := client.Resource(resource).Namespace(namespace).Delete(context.TODO(), unstructObj.GetName(), deleteOptions); err != nil {
		return err
	}

	return nil
}

func (in *Instancer) ListObjects(unstructObj *unstructured.Unstructured, namespace string) ([]string, error) {
	resource, err := in.GetObjectResource(unstructObj)
	if err != nil {
		return nil, err
	}

	client, err := dynamic.NewForConfig(in.k8sConfig)
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

// todo: query and cache gvr on init, not for every request
func (in *Instancer) GetObjectResource(unstructObj *unstructured.Unstructured) (schema.GroupVersionResource, error) {
	c, err := discovery.NewDiscoveryClientForConfig(in.k8sConfig)
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

func (in *Instancer) QueryInstancedChallenges(namespace string) (map[string]*template.Template, error) {
	resource := schema.GroupVersionResource{
		Group:    "k8s.maplebacon.org",
		Version:  "unstable",
		Resource: "instancedchallenges",
	}

	client, err := dynamic.NewForConfig(in.k8sConfig)
	if err != nil {
		return nil, err
	}

	chalList, err := client.Resource(resource).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret := make(map[string]*template.Template)

	for _, c := range chalList.Items {
		tmplStr, found, err := unstructured.NestedString(c.Object, "spec", "challengeTemplate")
		if err != nil || !found {
			fmt.Printf("template not found for challenge crd %v: error=%v", c.GetName(), err)
			continue
		}

		tmpl, err := template.New("challenge").Parse(tmplStr)
		if err != nil {
			in.log.Error().Err(err).Str("challenge", c.GetName()).Msg("could not parse a challenge template")
			continue
		}
		ret[c.GetName()] = tmpl
	}
	return ret, nil
}

func (in *Instancer) QueryInstancedChallenge(name string, namespace string) ([]unstructured.Unstructured, error) {
	resource := schema.GroupVersionResource{
		Group:    "k8s.maplebacon.org",
		Version:  "unstable",
		Resource: "instancedchallenges",
	}

	client, err := dynamic.NewForConfig(in.k8sConfig)
	if err != nil {
		return nil, err
	}

	chal, err := client.Resource(resource).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	resources, found, err := unstructured.NestedSlice(chal.Object, "spec", "resources")
	if err != nil || !found {
		fmt.Printf("resources not found for challenge crd %v: error=%v", chal.GetName(), err)
		return nil, err
	}
	res := make([]unstructured.Unstructured, 0)
	for _, r := range resources {
		obj, ok := r.(map[string]interface{})
		if !ok {
			fmt.Printf("could not parse object")
		}
		res = append(res, unstructured.Unstructured{Object: obj})
	}

	return res, nil
}
