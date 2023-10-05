package k8s_test

import (
	"fmt"
	"log"

	"github.com/ubcctf/instanced/k8s"
)

func ExampleUnmarshalSingleManifest() {
	manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-instance-test-deployment
  namespace: challenges
spec:
  selector:
  matchLabels:
    app: nginx-instance-test
  replicas: 1
  template:
    metadata:
      labels:
    app: nginx-instance-test
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
`
	obj, err := k8s.UnmarshalSingleManifest(manifest)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Println(obj)
	// Output: &{map[apiVersion:apps/v1 kind:Deployment metadata:map[name:nginx-instance-test-deployment namespace:challenges] spec:map[matchLabels:map[app:nginx-instance-test] replicas:1 selector:<nil> template:map[app:nginx-instance-test metadata:map[labels:<nil>] spec:map[containers:[map[image:nginx:1.14.2 name:nginx ports:[map[containerPort:80]]]]]]]]}
}

func ExampleUnmarshalManifestFile() {
	manifest := `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-instance-test-deployment
  namespace: challenges
spec:
  selector:
    matchLabels:
      app: nginx-instance-test
  replicas: 1
  template:
    metadata:
      labels:
        app: nginx-instance-test
    spec:
      containers:
        - name: nginx
          image: nginx:1.14.2
          ports:
            - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-instance-test
  namespace: challenges
spec:
  selector:
    app: nginx-instance-test
  ports:
    - name: http
      protocol: TCP
      port: 8080
      targetPort: 80
---
`
	objs, err := k8s.UnmarshalManifestFile(manifest)
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, v := range objs {
		if v.GetName() != "" {
			fmt.Println(v.GetKind(), v.GetName())
		}
	}
	// Output: Deployment nginx-instance-test-deployment
	// Service nginx-instance-test
}

func ExampleUnmarshalChallenges() {
	chals := make(map[string]string, 2)
	chals["nginx"] = `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-instance-test-deployment
  namespace: challenges
spec:
  selector:
    matchLabels:
      app: nginx-instance-test
  replicas: 1
  template:
    metadata:
      labels:
        app: nginx-instance-test
    spec:
      containers:
        - name: nginx
          image: nginx:1.14.2
          ports:
            - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-instance-test
  namespace: challenges
spec:
  selector:
    app: nginx-instance-test
  ports:
    - name: http
      protocol: TCP
      port: 8080
      targetPort: 80
---
`
	chalMap, err := k8s.UnmarshalChallenges(chals)
	if err != nil {
		log.Fatal(err)
	}
	obj, ok := chalMap["nginx"]
	if !ok {
		log.Fatal("key does not exist")
	}
	fmt.Println(obj)
	// Output: [{map[apiVersion:apps/v1 kind:Deployment metadata:map[name:nginx-instance-test-deployment namespace:challenges] spec:map[replicas:1 selector:map[matchLabels:map[app:nginx-instance-test]] template:map[metadata:map[labels:map[app:nginx-instance-test]] spec:map[containers:[map[image:nginx:1.14.2 name:nginx ports:[map[containerPort:80]]]]]]]]} {map[apiVersion:v1 kind:Service metadata:map[name:nginx-instance-test namespace:challenges] spec:map[ports:[map[name:http port:8080 protocol:TCP targetPort:80]] selector:map[app:nginx-instance-test]]]}]
}
