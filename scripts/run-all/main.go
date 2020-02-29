package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// Walk down the path, look for "workflow.yaml".
// If found, run `valet ensure -f workflow.yaml` from that directory.
func main() {
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.Base(path) == "workflow.yaml" {
			cmd := exec.Command("valet", "ensure", "-f", "workflow.yaml")
			cmd.Dir = filepath.Dir(path)

			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Fatalf("Error: %v\n\n%s", err, out)
			}
			log.Printf("Successfully processed %s\n", path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
