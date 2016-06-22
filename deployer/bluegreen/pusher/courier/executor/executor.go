package executor

import (
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/afero"
)

func New(fileSystem *afero.Afero) (Executor, error) {
	tempDir, err := fileSystem.TempDir("", "deployadactyl-")
	if err != nil {
		return Executor{}, err
	}

	return Executor{
		fileSystem: fileSystem,
		tempDir:    tempDir,
	}, nil
}

type Executor struct {
	tempDir    string
	fileSystem *afero.Afero
}

func (e Executor) Execute(args ...string) ([]byte, error) {
	command := exec.Command("cf", args...)
	command.Env = setEnv(os.Environ(), "CF_HOME", e.tempDir)
	return command.CombinedOutput()
}

func (e Executor) ExecuteInDirectory(directory string, args ...string) ([]byte, error) {
	command := exec.Command("cf", args...)
	command.Env = setEnv(os.Environ(), "CF_HOME", e.tempDir)
	command.Dir = directory
	return command.CombinedOutput()
}

func (e Executor) CleanUp() error {
	return e.fileSystem.RemoveAll(e.tempDir)
}

func setEnv(env []string, key, value string) []string {
	keyValuePair := key + "=" + value

	for i, envVar := range env {
		if strings.HasPrefix(envVar, key+"=") {
			env[i] = keyValuePair
			return env
		}
	}

	return append(env, keyValuePair)
}
