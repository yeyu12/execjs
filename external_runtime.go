package execjs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

type ExternalRuntime struct {
	name          string
	command       []string
	runner_source string
	encoding      string
	tempfile      bool
	available     bool
	binary_cache  []string
}

func BuildExternalRuntime(name string, command []string, runner_source string) *ExternalRuntime {
	r := &ExternalRuntime{name: name, command: command, runner_source: runner_source}
	r.available = (r.binary() != nil)
	return r
}

func (r *ExternalRuntime) Exec_(source string) (interface{}, error) {
	return r.Compile("").Exec_(source)
}

func (r *ExternalRuntime) Eval(source string) (interface{}, error) {
	return r.Compile("").Eval(source)
}

func (r *ExternalRuntime) Compile(source string) RuntimeContextInterface {
	if r.Is_available() {
		return r.compile(source)
	} else {
		return nil
	}
}

func (r *ExternalRuntime) Is_available() bool {
	return r.available
}

func (r *ExternalRuntime) compile(source string) RuntimeContextInterface {
	return &Context{runtime: r, source: source, tempfile: r.tempfile}
}

func (r *ExternalRuntime) binary() []string {
	if r.binary_cache == nil {
		r.binary_cache = which(r.command)
	}
	return r.binary_cache
}

type Context struct {
	runtime  *ExternalRuntime
	source   string
	cwd      string
	tempfile bool
}

func (c *Context) Exec_(source string) (interface{}, error) {
	if !c.Is_available() {
		return "", RuntimeUnavailableError{Message: fmt.Sprintf("runtime is not available on this system")}
	}
	output, err := c.exec_(source)
	if err != nil {
		return "", err
	}
	return output, nil
}

func (c *Context) Eval(source string) (interface{}, error) {
	if !c.Is_available() {
		return "", RuntimeUnavailableError{Message: fmt.Sprintf("runtime is not available on this system")}
	}
	output, err := c.eval(source)
	if err != nil {
		return "", err
	}
	return output, nil
}

func (c *Context) Call(name string, args ...interface{}) (interface{}, error) {
	if !c.Is_available() {
		return "", RuntimeUnavailableError{Message: fmt.Sprintf("runtime is not available on this system")}
	}

	output, err := c.call(name, args...)
	if err != nil {
		return "", err
	}
	return output, nil
}

func (c *Context) Is_available() bool {
	return c.runtime.Is_available()
}

func (c *Context) eval(source string) (interface{}, error) {
	var data string
	if len(strings.TrimSpace(source)) == 0 {
		data = "''"
	} else {
		data = "'('+'" + source + "'+')'"
	}
	code := fmt.Sprintf("return eval(%s)", data)
	return c.Exec_(code)
}

func (c *Context) exec_(source string) (interface{}, error) {
	if c.source != "" {
		source = c.source + "\n" + source
	}
	var (
		output string
		err    error
	)
	if c.tempfile {
		output, err = c.exec_with_tempfile(source)
		if err != nil {
			return "", err
		}
	} else {
		output, err = c.exec_with_pipe(source)
		if err != nil {
			return "", err
		}
	}
	return c.extract_result(output)
}

func (c *Context) call(identifier string, args ...interface{}) (interface{}, error) {
	stringArgs := make([]string, 0)
	for _, v := range args {
		arg, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		stringArgs = append(stringArgs, string(arg))
	}
	argString := strings.Join(stringArgs, ",")
	output, err := c.eval(fmt.Sprintf("%s.apply(this,[%s])", identifier, argString))
	if err != nil {
		return "", err
	}
	return output, nil
}

func (c *Context) exec_with_tempfile(source string) (string, error) {
	return "", nil
}

func (c *Context) exec_with_pipe(source string) (string, error) {
	binary := c.runtime.binary()
	cmd := exec.Command(binary[0], binary[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	input := c.compile(source)
	_, err = stdin.Write([]byte(input))
	if err != nil {
		return "", err
	}
	stdin.Close()
	err = cmd.Wait()
	outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
	if err != nil {
		return errStr, err
	}
	return outStr, nil
}

func (c *Context) extract_result(output string) (interface{}, error) {
	ret := strings.Split(output, "\n")
	data := ret[len(ret)-2]
	var res []interface{}
	json.Unmarshal([]byte(data), &res)
	if len(res) == 1 {
		if res[0].(string) == "ok" {
			return "", nil
		} else {
			return "", ProgramError{}
		}
	} else {
		return res[1], nil
	}
}

func (c *Context) compile(source string) string {
	runner_source := c.runtime.runner_source
	return strings.Replace(runner_source, "#{source}", source, 1)
}

func which(command []string) []string {
	name := command[0]
	path := find_executable(name)
	binary := make([]string, len(command))
	if path != "" {
		copy(binary, command)
		binary[0] = path
		return binary
	} else {
		return nil
	}
}

func find_executable(prog string) string {
	pathlist := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	filename := ""

	for _, dir := range pathlist {
		if dir == "" {
			continue
		}

		filename = path.Join(dir, prog)
		if runtime.GOOS == "windows" {
			filename += ".exe"
		}
		info, err := os.Stat(filename)
		if err != nil {
			continue
		}

		if runtime.GOOS == "windows" {
			ext := strings.ToLower(filepath.Ext(filename))
			switch ext {
			case ".exe", ".com", ".bat", ".cmd", ".ps1":
				return filename
			}
		} else {
			if info.Mode()&0111 == 0111 {
				return filename
			}
		}
	}

	return filename
}

func NodeCommand() *ExternalRuntime {
	r := node_node()
	if r.Is_available() {
		return r
	}
	return node_nodejs()
}

func node_node() *ExternalRuntime {
	return BuildExternalRuntime("Node.js (V8)", []string{"node"}, Node_source)
}

func node_nodejs() *ExternalRuntime {
	return BuildExternalRuntime("Node.js (V8)", []string{"nodejs"}, Node_source)
}
