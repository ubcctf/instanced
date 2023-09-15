package main

import (
	"log"
	"time"

	"github.com/google/uuid"
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

func ExampleUnmarshalChallenges() {
	log.Println("Running: ExampleUnmarshalChallenges")
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
	chalMap, err := UnmarshalChallenges(chals)
	if err != nil {
		log.Fatal(err)
	}
	obj, ok := chalMap["nginx"]
	if !ok {
		log.Fatal("key does not exist")
	}
	log.Println(obj)
	// Output:
}

func ExampleGetChalObjsFromTemplate() {
	log.Println("Running: ExampleGetChalObjsFromTemplate")
	in := Instancer{}
	in.config = &Config{}
	in.config.Challenges = make(map[string]string, 2)
	in.config.Challenges["nginx"] = `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-instance-{{.ID}}-deployment
  namespace: challenges
spec:
  selector:
    matchLabels:
      app: nginx-instance-{{.ID}}
  replicas: 1
  template:
    metadata:
      labels:
        app: nginx-instance-{{.ID}}
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
  name: nginx-instance-{{.ID}}
  namespace: challenges
spec:
  selector:
    app: nginx-instance-{{.ID}}
  ports:
    - name: http
      protocol: TCP
      port: 8080
      targetPort: 80
---
`
	in.ParseTemplates()
	chal, err := in.GetChalObjsFromTemplate("nginx", uuid.NewString()[0:8])
	if err != nil {
		log.Fatal(err)
	}
	log.Println(len(chal))
	log.Println(chal)
	// Output:
}

func ExampleInsertInstanceRecord() {
	in := Instancer{}
	in.InitDB("./tmp/test.db")
	in.InsertInstanceRecord(time.Hour, "1", "test_challenge", "abcdefg")
	recs, err := in.ReadInstanceRecords()
	if err != nil {
		log.Fatal(err)
	}
	for _, v := range recs {
		log.Println(v)
	}
	// Output:
}
