package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Disk struct {
	Path string
}

func MakeDisk(path string) (*Disk, error) {
	err := createFolder(path)

	fmt.Printf("Folder to store artifacts: %s", path)

	if err != nil {
		return nil, err
	}

	return &Disk{Path: path}, nil
}

func createFolder(p string) error {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		err := os.MkdirAll(p, 0755)

		return err
	}

	return nil
}

func (s *Disk) Put(hash, team string, reader io.ReadCloser) error {
	b, err := io.ReadAll(reader)

	if err != nil {
		return err
	}

	teamfolder := filepath.Join(s.Path, team)

	err = createFolder(teamfolder)

	if err != nil {
		return err
	}

	p := filepath.Join(teamfolder, hash)

	return os.WriteFile(p, b, 0644)
}

func (s *Disk) Get(hash, team string) ([]byte, error) {
	p := filepath.Join(s.Path, team, hash)

	return os.ReadFile(p)
}
