package pcifunction

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
)

func isFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readUintFromFile(path string) (uint, error) {
	data, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return 0, errors.Wrapf(err, "unable to locate file: %v", path)
	}

	value, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, errors.Wrapf(err, "unable to convert string to int: %v", string(data))
	}

	return uint(value), nil
}

func evalSymlinkAndGetBaseName(path string) (string, error) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return "", errors.Wrapf(err, "error getting info about specified file: %s", path)
	}
	if fileInfo.Mode()&os.ModeSymlink == 0 {
		return "", errors.Errorf("specified file is not a symbolic link: %s", path)
	}

	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", errors.Wrapf(err, "error evaluating symbolic link: %s", path)
	}

	realPathBase := filepath.Base(realPath)

	return realPathBase, nil
}
