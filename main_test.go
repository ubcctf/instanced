package main

import (
	"log"
)

func ExampleUnmarshalSingleManifest() {
	log.Println("Running: ExampleUnmarshalSingleManifest")
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
	obj, err := UnmarshalSingleManifest(manifest)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println(obj)
	// Output:
}

func ExampleUnmarshalManifestFile() {
	log.Println("Running: ExampleUnmarshalManifestFile")
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
	objs, err := UnmarshalManifestFile(manifest)
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, v := range objs {
		if v.GetName() != "" {
			log.Println(v.GetKind(), v.GetName())
		}
	}
	// Output:
}
