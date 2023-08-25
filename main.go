package main

import "log"

func main() {
	instancer, err := InitInstancer()
	if err != nil {
		log.Fatal("failed to initialize")
	}

	instancer.Start()
}
