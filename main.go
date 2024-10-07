// main executable.
package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/most-1995/mediamtx/internal/core"
)

func main() {
	s, ok := core.New(os.Args[1:])
	if !ok {
		os.Exit(1)
	}
	s.Wait()

}

type StorePath struct {
	Name string `json:"name"`
}

func SavePathToJsonFile(name string) string {
	storePath := StorePath{
		Name: name,
	}

	jsonData, err := json.Marshal(storePath)

	if err != nil {
		return ""
	}

	_ = ioutil.WriteFile("test.json", jsonData, 0644)

	return ""
}
