package saver

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/defectus/glutton/pkg/iface"
	"github.com/stretchr/testify/assert"
)

func TestSimpleFileSystemSaverConfigureCreatesOutputFolder(t *testing.T) {
	root := filepath.Join(t.TempDir(), "nested", "out")
	s := new(SimpleFileSystemSaver)
	assert.NoError(t, s.Configure(&iface.Settings{OutputFolder: root, BaseName: "glutton_%d"}))

	info, err := os.Stat(root)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestSimpleFileSystemSaverSave(t *testing.T) {
	root := filepath.Join(t.TempDir(), "out")
	s := new(SimpleFileSystemSaver)
	assert.NoError(t, s.Configure(&iface.Settings{OutputFolder: root, BaseName: "glutton_%d"}))

	err := s.Save(&iface.PayloadRecord{
		Payload:   "hello world",
		Timestamp: time.Now(),
		Remote:    "127.0.0.1",
	})
	assert.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(root, "glutton_1"))
	assert.NoError(t, err)
	assert.Contains(t, string(content), "hello world")
}

func TestSimpleFileSystemSaverConfigureFailsOnInvalidFolder(t *testing.T) {
	// a file where a directory is expected makes MkdirAll fail
	file := filepath.Join(t.TempDir(), "not-a-dir")
	assert.NoError(t, os.WriteFile(file, []byte("x"), 0644))

	s := new(SimpleFileSystemSaver)
	err := s.Configure(&iface.Settings{OutputFolder: filepath.Join(file, "out"), BaseName: "glutton_%d"})
	assert.Error(t, err)
}
