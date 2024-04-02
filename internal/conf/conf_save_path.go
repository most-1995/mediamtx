package conf

import (
	"encoding/json"
	"io/ioutil"
)

type StorePath struct {
	Name         string        `json:"name"`
	OptionalPath *OptionalPath `json:"optional_path"`
}

func (conf *Conf) SavePathToJsonFile(name string, p *OptionalPath) string {

	arrStorePath := []StorePath{}

	data, err := ioutil.ReadFile("save_path.json")

	if err == nil {
		err = json.Unmarshal(data, &arrStorePath)

		if err != nil {
			return ""
		}
	}

	storePath := StorePath{
		Name:         name,
		OptionalPath: p,
	}

	arrStorePath = append(arrStorePath, storePath)

	jsonData, err := json.Marshal(arrStorePath)

	if err != nil {
		return ""
	}

	_ = ioutil.WriteFile("save_path.json", jsonData, 0644)

	return ""
}

func (conf *Conf) ReadPathFromJsonFile() map[string]*OptionalPath {
	data, err := ioutil.ReadFile("save_path.json")
	if err != nil {
		return nil
	}

	storePath := []StorePath{}

	err = json.Unmarshal(data, &storePath)

	if err != nil {
		return nil
	}

	mapPath := map[string]*OptionalPath{}

	for _, path := range storePath {
		mapPath[path.Name] = path.OptionalPath
	}

	return mapPath

}
