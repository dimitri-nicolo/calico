package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

type SwaggerDoc struct {
	Version string                 `json:"swagger"`
	Info    SwaggerDocInfo         `json:"info"`
	Paths   map[string]interface{} `json:"paths"`
}

type SwaggerDocInfo struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

const usage = "usage: swaggerapi <file> <target>"

var calicoFilter bool
var k8sFilter bool

func init() {
	flag.BoolVar(&calicoFilter, "c", false, "Show short form of calico api")
	flag.BoolVar(&k8sFilter, "k", false, "Show short form of k8s api")
}

// Remove non-calico entries from a generated swagger file
func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "%s\n", usage)
		os.Exit(1)
	}

	// Extract the JSON blob from the JSON file.
	swaggerb, err := ioutil.ReadFile(args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Change the JSON blob into an object.
	swaggerDoc := &SwaggerDoc{}
	err = json.Unmarshal(swaggerb, swaggerDoc)
	if err != nil {
		fmt.Printf("Failed at Unmarshalling: %s\n", err)
		os.Exit(1)
	}

	k8s := []string{}
	calico := []string{}
	// Remove the all the paths that are not projectcalico related.
	for path := range swaggerDoc.Paths {
		if !strings.Contains(path, "apis/projectcalico.org") {
			delete(swaggerDoc.Paths, path)
			k8s = append(k8s, path)
		} else {
			calico = append(calico, path)
		}
	}

	// Write back to a file.
	calicoSwagger, err := json.MarshalIndent(swaggerDoc, "", "  ")
	if err != nil {
		fmt.Printf("Failed at Marshalling: %s\n", err)
		os.Exit(1)
	}

	mode := int(0644)
	err = ioutil.WriteFile(args[1], calicoSwagger, os.FileMode(mode))
	if err != nil {
		fmt.Printf("Failed to write file: %s\n", err)
		os.Exit(1)
	}

	// Prints out Calico APIs if -c is set
	if calicoFilter {
		sort.Strings(calico)
		fmt.Println("Calico API:")
		for _, s := range calico {
			fmt.Println(s)
		}
	}
	// Prints out Kubernetes APIs if -k is set
	if k8sFilter {
		sort.Strings(k8s)
		fmt.Println("Kubernetes API:")
		for _, s := range k8s {
			fmt.Println(s)
		}
	}
}
