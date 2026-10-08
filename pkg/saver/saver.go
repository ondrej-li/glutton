package saver

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/defectus/glutton/pkg/iface"
	"github.com/pkg/errors"
)

// SimpleFileSystemSaver saves request to filesystem.
type SimpleFileSystemSaver struct {
	root     string
	basename string
	counter  int64
	debug    bool
}

// Save saves payload (request) to configured filesystem destination.
func (s *SimpleFileSystemSaver) Save(payload *iface.PayloadRecord) error {
	index := atomic.AddInt64(&s.counter, 1)
	if s.debug {
		log.Printf("SimpleFileSystemSaver_Save: output file %s", s.filename(index))
	}
	f, err := os.OpenFile(s.filename(index), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrapf(err, "error opening outfile %s", s.filename(index))
	}
	_, err = f.WriteString(payload.String())
	if err != nil {
		return errors.Wrapf(err, "error writing to outfile %s", s.filename(index))
	}
	err = f.Close()
	return errors.Wrapf(err, "error closing output %s", payload.String())
}

// Configure bootstraps the SimpleFileSystemSaver
// Namely the following params are used:
// * OutputFolder
// * BaseName - first part of the name
func (s *SimpleFileSystemSaver) Configure(settings *iface.Settings) error {
	s.root = settings.OutputFolder
	s.basename = settings.BaseName
	s.debug = settings.Debug
	if err := os.MkdirAll(s.root, 0755); err != nil {
		return errors.Wrapf(err, "error creating output folder %s", s.root)
	}
	return nil
}

func (s *SimpleFileSystemSaver) filename(index int64) string {
	return fmt.Sprintf(filepath.Join(s.root, s.basename), index)
}
