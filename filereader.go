package cmdexec

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

type fileFormat struct {
	Name           string
	Version        string
	DefaultTimeout int       "default_timeout"
	Commands       []Command "commands"
}

type fileCache map[string]struct {
	file fileFormat
	time int64
}

var filecache = make(fileCache)

func readFile(file string) fileFormat {
	var filedata fileFormat
	result, err := getCache(file)
	if err != nil {
		// cache does not exist. read config file
		source, err := ioutil.ReadFile(file)
		if err != nil {
			panic(err)
		}

		err = parseYAML(source, &filedata)
		if err != nil {
			panic(err)
		}

		result = saveCache(file, filedata)
	}
	return result
}

func getCache(file string) (fileFormat, error) {
	if filecache[file].time <= time.Now().Unix()-fileCacheTime {
		return fileFormat{}, errors.New("Cache expired")
	} else {
		return filecache[file].file, nil
	}
}

func saveCache(file string, filedata fileFormat) fileFormat {
	var tmp = filecache[file]
	tmp.file = filedata
	tmp.time = time.Now().Unix()
	filecache[file] = tmp
	return filedata
}

func parseYAML(source []byte, output interface{}) (err error) {
	return yaml.Unmarshal(source, output)
}

func parseJSON(source []byte, output interface{}) (err error) {
	return json.Unmarshal(source, output)
}
