package main

import (
	"log"
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
	obj, err := UnmarshalSingleManifest(manifest)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println(obj.Object)
	// Output:
}
