package main

import (
    "github.com/minio/minio-go"
	"log"
	"os"
	"io"
	"errors"
	"time"
)

type ControlFile struct {
    name string
    contents string
}

type Bucket struct {
    minioClient *minio.Client
    Name string
    //ctlFile ControlFile
    files map[string]interface{}
}

func (b *Bucket) ReadAt(p []byte, off int64) (int, error) {
    return 0, io.EOF
}

func (b *Bucket) WriteAt(p []byte, off int64) (int, error) {
    return 0, errors.New("can't write in directories")
}

func (b *Bucket) Close() error {
	return nil
}

func (b *Bucket) size() int64 {
    return 0
}

func (b *Bucket) ModTime() time.Time {
	return time.Now().Truncate(time.Hour)
}

func (b *Bucket) IsDir() bool {
	return true
}

func (b *Bucket) Mode() os.FileMode {
    return os.ModeDir | 0755
}

func (b *Bucket) Size() int64 {
	return b.size()
}

func (b *Bucket) Open() *dir {
    c := make(chan stat, 10)
	done := make(chan struct{})
    go func() {
        close(c)
    }()

    log.Printf("returning a bucket")
	return &dir{
		c:    c,
		done: done,
	}
}

type BucketList struct {
    minioClient *minio.Client
    //ctlFile ControlFile
    Buckets map[string]interface{}
}

func (b *BucketList) ReadAt(p []byte, off int64) (int, error) {
    return 0, io.EOF
}

func (b *BucketList) WriteAt(p []byte, off int64) (int, error) {
    return 0, errors.New("can't write in directories")
}

func (b *BucketList) Close() error {
	return nil
}

func (b *BucketList) size() int64 {
    return 0
}

func (b *BucketList) ModTime() time.Time {
	return time.Now().Truncate(time.Hour)
}

func (b *BucketList) IsDir() bool {
	return true
}

func (b *BucketList) Mode() os.FileMode {
    return os.ModeDir | 0755
}

func (b *BucketList) Size() int64 {
	return b.size()
}

func (b *BucketList) Open() *dir {
    log.Printf("in bucketmkdir")

    c := make(chan stat, 10)
	done := make(chan struct{})
    go func() {
        b.Buckets = make(map[string]interface{})
        buckets, _ := b.minioClient.ListBuckets()
        log.Printf("got %d buckets", len(buckets))
        LoopMap:
        for _, bucket := range buckets {
            bucket := &Bucket{Name: bucket.Name}
            b.Buckets[bucket.Name] = bucket
            select {
            case c <- stat{name: bucket.Name, file: bucket}:
                log.Printf("writing a bucket")
            case <-done:
                log.Printf("got a done")
                break LoopMap
            }
        }

        //c <- stat{name: "_cli", file: &fakefile{v: b.ctlFile}}
        log.Printf("finished the loop")
		close(c)
	}()

    log.Printf("returning a directory")
	return &dir{
		c:    c,
		done: done,
	}
}

