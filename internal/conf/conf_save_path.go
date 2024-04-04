package conf

import (
	"encoding/json"
	"os"
)

type StorePath struct {
	Name         string        `json:"name"`
	OptionalPath *OptionalPath `json:"optional_path"`
}

func (conf *Conf) SavePathToJsonFile(name string, p *OptionalPath) string {

	arrStorePath := []StorePath{}

	data, err := os.ReadFile("./saved_path/save_path.json")

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

	_ = os.WriteFile("./saved_path/save_path.json", jsonData, 0644)

	return ""
}

func (conf *Conf) ReadPathFromJsonFile() map[string]*OptionalPath {
	data, err := os.ReadFile("./saved_path/save_path.json")
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

func (conf *Conf) DeletePathFromJsonFile(name string) string {
	data, err := os.ReadFile("./saved_path/save_path.json")
	if err != nil {
		return ""
	}

	storePath := []StorePath{}

	err = json.Unmarshal(data, &storePath)

	if err != nil {
		return ""
	}

	for i, path := range storePath {
		if path.Name == name {
			storePath = append(storePath[:i], storePath[i+1:]...)
			break
		}
	}

	jsonData, err := json.Marshal(storePath)

	if err != nil {
		return ""
	}

	_ = os.WriteFile("./saved_path/save_path.json", jsonData, 0644)

	return ""
}

func (conf *Conf) UpdatePathFromJsonFile(name string, p *OptionalPath) string {
	data, err := os.ReadFile("./saved_path/save_path.json")
	if err != nil {
		return ""
	}

	storePath := []StorePath{}

	err = json.Unmarshal(data, &storePath)

	if err != nil {
		return ""
	}

	for i, path := range storePath {
		if path.Name == name {
			storePath[i].OptionalPath = p
			break
		}
	}

	jsonData, err := json.Marshal(storePath)

	if err != nil {
		return ""
	}

	_ = os.WriteFile("./saved_path/save_path.json", jsonData, 0644)

	return ""
}
