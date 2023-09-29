.PHONY: test build

test:
	mkdir -p tmp
	go test
	rm -r tmp

build:
	GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o ./out/instanced

docker:
	docker build . --tag us.gcr.io/maplectf/instanced:latest && docker push us.gcr.io/maplectf/instanced:latest