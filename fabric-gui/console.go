package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Fabric GUI Console App")
	fmt.Println("Go version:", os.Getenv("GOVERSION"))
	dir, _ := os.Getwd()
	fmt.Println("Working directory:", dir)
	fmt.Println("Environment variables:")
	for _, env := range os.Environ() {
		fmt.Println(env)
	}
}