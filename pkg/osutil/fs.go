package osutil

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// DirPerms specify directory permissions.
const DirPerms = os.FileMode(0755)

// FilePerms specify file permissions.
const FilePerms = os.FileMode(0644)

// CopyFile copies file in arbitrary filesystem structure.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("could not open: %s", err)
	}
	defer in.Close()

	inStat, err := in.Stat()
	if err != nil {
		return fmt.Errorf("could not get src stat: %s", err)
	}
	if !inStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	err = os.MkdirAll(filepath.Dir(dst), DirPerms)
	if err != nil {
		return fmt.Errorf("could not prepare dest dir: %s", err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("could not create dest file: %s", err)
	}

	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("copy error: %s", err)
	}

	err = out.Sync()
	if err != nil {
		return fmt.Errorf("FS sync error: %s", err)
	}

	err = out.Close()
	if err != nil {
		return fmt.Errorf("dest file close error: %s", err)
	}

	return nil
}

// WriteFile writes data to path ensuring parent directory exists
// and correct permissions.
func WriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), DirPerms); err != nil {
		return err
	}

	if err := ioutil.WriteFile(path, data, FilePerms); err != nil {
		return err
	}

	return nil
}

// DirExists checks if given path exists and is a directory.
func DirExists(path string) error {
	fi, err := os.Stat(path)

	switch {
	case os.IsNotExist(err):
		return fmt.Errorf("does not exist: %s", path)
	case err != nil:
		return err
	case !fi.IsDir():
		return fmt.Errorf("not directory: %s", path)
	}

	return nil
}

// FileExists checks if given path exists and is a file.
func FileExists(path string) error {
	fi, err := os.Stat(path)

	switch {
	case os.IsNotExist(err):
		return fmt.Errorf("does not exist: %s", path)
	case err != nil:
		return err
	case !fi.Mode().IsRegular():
		return fmt.Errorf("not regular file: %s", path)
	}

	return nil
}
