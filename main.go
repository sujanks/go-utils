package main

import (
	"fmt"
	"github.com/sujanks/go-cql/src/yaml"
	"os"
)

func main() {
	tree, _ := yaml.LoadEncryptedFile("./src/yaml/test.yaml")
	yaml.DecryptTree(tree)
	content, err := yaml.EmitPlainFile(tree)
	if err != nil {
		fmt.Errorf("error file emit content %v", err)
	}
	file, err := os.Create("./src/yaml/plain.yaml")
	if err != nil {
		fmt.Errorf("error %v", err)
	}
	defer file.Close()
	file.Write(content)
}
