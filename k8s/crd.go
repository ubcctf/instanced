package k8s

import (
	"context"
	"fmt"

	"strings"

	"text/template"

	"github.com/rs/zerolog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
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

func (k *KubeClient) QueryInstancedChallenges(ctx context.Context, namespace string) (map[string]*template.Template, error) {
	log := zerolog.Ctx(ctx)
	resource := schema.GroupVersionResource{
		Group:    "k8s.maplebacon.org",
		Version:  "unstable",
		Resource: "instancedchallenges",
	}

	client, err := dynamic.NewForConfig(k.Config)
	if err != nil {
		return nil, err
	}

	chalList, err := client.Resource(resource).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret := make(map[string]*template.Template)

	for _, c := range chalList.Items {
		hidden, found, err := unstructured.NestedBool(c.Object, "spec", "hidden")
		if err != nil || found && hidden {
			log.Info().Err(err).Str("challenge", c.GetName()).Msg("skipping hidden challenge")
			continue
		}
		tmplStr, found, err := unstructured.NestedString(c.Object, "spec", "challengeTemplate")
		if err != nil || !found {
			fmt.Printf("template not found for challenge crd %v: error=%v", c.GetName(), err)
			continue
		}

		tmpl, err := template.New("challenge").Parse(tmplStr)
		if err != nil {
			log.Error().Err(err).Str("challenge", c.GetName()).Msg("could not parse a challenge template")
			continue
		}
		ret[c.GetName()] = tmpl
	}
	return ret, nil
}

func (k *KubeClient) QueryInstancedChallenge(ctx context.Context, name string, namespace string) ([]unstructured.Unstructured, error) {
	resource := schema.GroupVersionResource{
		Group:    "k8s.maplebacon.org",
		Version:  "unstable",
		Resource: "instancedchallenges",
	}

	client, err := dynamic.NewForConfig(k.Config)
	if err != nil {
		return nil, err
	}

	chal, err := client.Resource(resource).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
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
