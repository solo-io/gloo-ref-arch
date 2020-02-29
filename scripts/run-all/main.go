package main

import (
	"bufio"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
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
			if err := Run("ensure", filepath.Dir(path)); err != nil {
				return err
			}
			if err := Run("teardown", filepath.Dir(path)); err != nil {
				return err
			}
			log.Printf("Successfully processed %s\n", path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

}

func Run(valetCmd, dir string) error {
	if handler, err := Stream(valetCmd, dir); err != nil {
		return err
	} else if err := handler.StreamHelper(errors.Errorf("error running valet workflow")); err != nil {
		return err
	}
	return nil
}

type CommandStreamHandler struct {
	WaitFunc func() error
	Stdout   io.Reader
	Stderr   io.Reader
}

func (c *CommandStreamHandler) StreamHelper(inputErr error) error {
	go func() {
		stdoutScanner := bufio.NewScanner(c.Stdout)
		for stdoutScanner.Scan() {
			log.Println(stdoutScanner.Text())
		}
		if err := stdoutScanner.Err(); err != nil {
			log.Println("reading stdout from current command context:", err)
		}
	}()
	stderr, _ := ioutil.ReadAll(c.Stderr)
	if err := c.WaitFunc(); err != nil {
		log.Println(fmt.Sprintf("%s\n", stderr))
		return inputErr
	}
	return nil
}

func Stream(valetCmd, dir string) (*CommandStreamHandler, error) {
	cmd := exec.Command("valet", valetCmd, "-f", "workflow.yaml")
	cmd.Dir = dir
	outReader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	errReader, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	return &CommandStreamHandler{
		WaitFunc: func() error {
			return cmd.Wait()
		},
		Stdout: outReader,
		Stderr: errReader,
	}, nil
}
