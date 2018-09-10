package main

import (
	"bytes"
	"errors"
	"fmt"
    //"regexp"
    "log"
	"io"
	"os"
    "github.com/minio/minio-go"
    "time"
)

// Turn Go types into files

type statable interface {
    ModTime() time.Time

    IsDir() bool

    Mode() os.FileMode

    Size() int64
}

type fakefile struct {
	v      interface{}
	offset int64
	set    func(s string)
}

func (f *fakefile) ReadAt(p []byte, off int64) (int, error) {
	var s string
	if v, ok := f.v.(fmt.Stringer); ok {
		s = v.String()
	} else {
		s = fmt.Sprint(f.v)
	}
	if off > int64(len(s)) {
		return 0, io.EOF
	}
	n := copy(p, s)
	return n, nil
}

func (f *fakefile) WriteAt(p []byte, off int64) (int, error) {
	buf, ok := f.v.(*bytes.Buffer)
	if !ok {
		return 0, errors.New("not supported")
	}
	if off != f.offset {
		return 0, errors.New("no seeking")
	}
	n, err := buf.Write(p)
	f.offset += int64(n)
	return n, err
}

func (f *fakefile) Close() error {
	if f.set != nil {
		f.set(fmt.Sprint(f.v))
	}
	return nil
}

func (f *fakefile) size() int64 {
	switch f.v.(type) {
	case map[string]interface{}, []interface{}:
		return 0
	}
	return int64(len(fmt.Sprint(f.v)))
}

type stat struct {
	name string
	file statable
}

func (s *stat) Name() string     { return s.name }
func (s *stat) Sys() interface{} { return s.file }

func (s *stat) ModTime() time.Time {
	return time.Now().Truncate(time.Hour)
}

func (s *stat) IsDir() bool {
	return s.Mode().IsDir()
}

func (s *stat) Mode() os.FileMode {
	switch s.file.v.(type) {
	case map[string]interface{}, minio.BucketInfo:
		return os.ModeDir | 0755
	case []interface{}:
		return os.ModeDir | 0755
	}
	return 0644
}

func (s *stat) Size() int64 {
	return s.file.size()
}

type dir struct {
	c    chan stat
	done chan struct{}
}

func mkdir(val interface{}) *dir {
    c := make(chan stat, 10)
	done := make(chan struct{})
    //re := regexp.MustCompile("[[:^ascii:]]")

    log.Printf("returning an old directory")
	return &dir{
		c:    c,
		done: done,
	}
}

func (d *dir) Readdir(n int) ([]os.FileInfo, error) {
    log.Printf("Doing a readdir")
	var err error
	fi := make([]os.FileInfo, 0, 10)
	for i := 0; i < n; i++ {
        log.Printf("on entry %d", i)
		s, ok := <-d.c
		if !ok {
            log.Printf("something's not okay")
			err = io.EOF
			break
		}

        log.Printf("appending")
		fi = append(fi, &s)
	}
    log.Printf("returning")
	return fi, err
}

func (d *dir) Close() error {
	close(d.done)
	return nil
}
