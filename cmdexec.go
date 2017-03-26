package cmdexec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type fileFormat struct {
	Name           string
	Version        string
	DefaultTimeout int       "default_timeout"
	Commands       []command "commands"
}

type command struct {
	Command  string              "command"
	Required []map[string]string "required"
	Timeout  int                 "timeout"
}

type jsonResponse struct {
	Result map[string]string
}

type appError struct {
	Message string
	Code    int
}

type FileCache map[string]struct {
	file fileFormat
	time int64
}

var filecache = make(FileCache)

const fileCacheTime = 30   // seconds
const default_timeout = 10 // seconds

func RenderFile(file string, parameters map[string][]string, w http.ResponseWriter) {
	filedata := readFile(file)

	if err := checkRequiredParameters(filedata, parameters); err != nil {
		var errorData appError
		errorData.Message = err.Error()
		errorData.Code = 1
		w.Write([]byte(ResponseToJSON(errorData)))
		panic(err)
	}

	for _, cmd := range filedata.Commands {
		fmt.Printf("Running: %v\n", cmd)
		var args []string
		command := strings.Split(cmd.Command, " ")
		if len(command) > 1 {
			args = command[1:]
		}
		if cmd.Timeout == 0 {
			cmd.Timeout = default_timeout
		}
		w.Write([]byte(RunCommand(command[0], cmd.Timeout, args)))
	}
}

func ExecFile(file string, parameters map[string][]string) string {
	filedata := readFile(file)

	if err := checkRequiredParameters(filedata, parameters); err != nil {
		var errorData appError
		errorData.Message = err.Error()
		errorData.Code = 1
		return ResponseToJSON(errorData)
	}

	var returndata jsonResponse
	returndata.Result = make(map[string]string, len(filedata.Commands))
	for _, cmd := range filedata.Commands {
		fmt.Printf("Running: %v\n", cmd)
		var args []string
		command := strings.Split(cmd.Command, " ")
		if len(command) > 1 {
			args = command[1:]
		}
		if cmd.Timeout == 0 {
			cmd.Timeout = default_timeout
		}
		returndata.Result[command[0]] = RunCommand(command[0], cmd.Timeout, args)
	}

	return ResponseToText(returndata)
}

func RunCommand(cmd string, timeout int, args []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	result, err := exec.CommandContext(ctx, cmd, args...).Output()
	if err != nil {
		fmt.Printf("Error %v: %v. Res: %s \n", cmd, err, result)
	} else {
		//fmt.Printf("Command %v: Timeout: %d = %s\n", cmd, timeout, result)
	}
	fmt.Printf("Command %v: Timeout: %d = %s\n", cmd, timeout, result)
	return string(result[:])
}

func checkRequiredParameters(filedata fileFormat, parameters map[string][]string) (err error) {
	// check required params
	for index, cmd := range filedata.Commands {
		for _, req := range cmd.Required {
			for name, expr := range req {
				if len(parameters[name]) == 0 {
					return errors.New(fmt.Sprintf("Parameter %s is missing", name))
				} else {
					for _, value := range parameters[name] {
						re := regexp.MustCompile(expr)
						rexp := re.MatchString(value)
						if err != nil {
							return errors.New(fmt.Sprintf("Can not parse regexp '%s' for '%s'", expr, name))
						}
						if rexp != true {
							return errors.New(fmt.Sprintf("Value '%s' is not valid.", name))
						}
						filedata.Commands[index].Command = strings.Replace(cmd.Command, "{{"+name+"}}", value, -1)
					}
				}
			}
		}
	}
	return nil
}

func ResponseToText(response jsonResponse) string {
	text := ""
	for _, result := range response.Result {
		text += result
	}
	return text
}

func ResponseToJSON(response interface{}) string {
	encode, _ := json.Marshal(response)
	return string(encode)
}

func readFile(file string) fileFormat {
	var filedata fileFormat
	if filecache[file].time <= time.Now().Unix()-fileCacheTime {
		// cache does not exist. read config file
		source, err := ioutil.ReadFile(file)
		if err != nil {
			panic(err)
		}
		err = yaml.Unmarshal(source, &filedata)
		if err != nil {
			panic(err)
		}
		fmt.Printf("ReadFile %s: %v\n", file, filedata)

		var tmp = filecache[file]
		tmp.file = filedata
		tmp.time = time.Now().Unix()
		filecache[file] = tmp
	}
	return filecache[file].file
}
