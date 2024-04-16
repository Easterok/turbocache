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

func (d *Disk) Put(hash, team string, reader io.ReadCloser) error {
	b, err := io.ReadAll(reader)

	if err != nil {
		return err
	}

	teamfolder := filepath.Join(d.Path, team)

	err = createFolder(teamfolder)

	if err != nil {
		return err
	}

	p := filepath.Join(teamfolder, hash)

	return os.WriteFile(p, b, 0644)
}

func (d *Disk) Get(hash, team string) ([]byte, error) {
	p := filepath.Join(d.Path, team, hash)

	return os.ReadFile(p)
}

func saveToFile(name string, data []byte) error {
	file, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)

	if err != nil {
		return err
	}

	defer file.Close()

	_, err = file.Write(append(data, byte('\n')))

	if err != nil {
		return err
	}

	return nil
}

func (d *Disk) SaveEvent(team string, data []byte) error {
	teamfolder := filepath.Join(d.Path, team)

	err := createFolder(teamfolder)

	if err != nil {
		return err
	}

	return saveToFile(filepath.Join(teamfolder, "_events"), data)
}

func (d *Disk) SaveMeta(hash, team string, data []byte) error {
	teamfolder := filepath.Join(d.Path, team)

	err := createFolder(teamfolder)

	if err != nil {
		return err
	}

	return saveToFile(filepath.Join(teamfolder, "_meta"), data)
}

func (d *Disk) GetMeta(team string) ([]byte, error) {
	p := filepath.Join(d.Path, team, "_meta")

	return os.ReadFile(p)
}

func (d *Disk) GetEvents(team string) ([]byte, error) {
	p := filepath.Join(d.Path, team, "_events")

	return os.ReadFile(p)
}
