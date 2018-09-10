// Command jsonfs allows for the consumption and manipulation of a JSON
// object as a file system hierarchy.
package main

import (
	"bytes"
    "fmt"
    "github.com/minio/minio-go"
	"math"
	"flag"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

    "aqwari.net/net/styx"
)

var (
	addr    = flag.String("a", ":5640", "Port to listen on")
	debug   = flag.Bool("D", false, "trace 9P messages")
	verbose = flag.Bool("v", false, "print extra info")
)

type server struct {
	file BucketList
    client *minio.Client
}

var logrequests styx.HandlerFunc = func(s *styx.Session) {
	for s.Next() {
		log.Printf("%q %T %s", s.User, s.Request(), s.Request().Path())
	}
}

func main() {
	flag.Parse()
	log.SetPrefix("")
	log.SetFlags(0)
    // Initialize minio client object.
    ssl := true
    minioClient, err := minio.New("play.minio.io:9000", "Q3AM3UQ867SPQQA43P2F", "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG", ssl)
    if err != nil {
            fmt.Println(err)
            return
    }

    //file := make(map[string]interface{})
    file := BucketList{minioClient, ControlFile{"_9p_s3config", "play.minio.io:9000"}, make(map[string]interface{})}
    srv := server{file, minioClient}
	var styxServer styx.Server
	if *verbose {
		styxServer.ErrorLog = log.New(os.Stderr, "", 0)
	}
	if *debug {
		styxServer.TraceLog = log.New(os.Stderr, "", 0)
	}
	styxServer.Addr = *addr
	styxServer.Handler = styx.Stack(logrequests, &srv)
	styxServer.MaxSize = math.MaxInt64

	log.Fatal(styxServer.ListenAndServe())
}

func walkTo(v interface{}, loc string) (interface{}, interface{}, bool) {
	cwd := v
	parts := strings.FieldsFunc(loc, func(r rune) bool { return r == '/' })
	var parent interface{}
    log.Printf("walking %s", loc)

	for _, p := range parts {
		switch v := cwd.(type) {
		case BucketList:
            log.Printf("cwd.Type is BucketList")
			parent = v
			if child, ok := v.Buckets[p]; !ok {
				return nil, nil, false
			} else {
				cwd = child
			}
		case map[string]interface{}:
            log.Printf("cwd.Type is map[string]interface")
			parent = v
			if child, ok := v[p]; !ok {
				return nil, nil, false
			} else {
				cwd = child
			}
		case []interface{}:
            log.Printf("cwd.Type is []interface")
			parent = v
			i, err := strconv.Atoi(p)
			if err != nil {
				return nil, nil, false
			}
			if len(v) <= i {
				return nil, nil, false
			}
			cwd = v[i]
		default:
            log.Printf("cwd.Type is default")
			return nil, nil, false
		}
	}
    log.Printf("no parts")
	return parent, cwd, true
}

func (srv *server) Serve9P(s *styx.Session) {
	for s.Next() {
		t := s.Request()
		parent, file, ok := walkTo(srv.file, t.Path())
        log.Printf("cwd is %s", srv.file)

		if !ok {
            log.Printf("errored out on walkTo")
			t.Rerror("no such file or directory")
			continue
		}
		fi := &stat{name: path.Base(t.Path()), file: &fakefile{v: file}}
		switch t := t.(type) {
		case styx.Twalk:
			t.Rwalk(fi, nil)
		case styx.Topen:
			switch v := file.(type) {
            case BucketList:
                log.Printf("running bucket mkdir")
                t.Ropen(v.Open(), nil)
			case map[string]interface{}, []interface{}:
                log.Printf("running mkdir 1")
				t.Ropen(mkdir(v), nil)
			default:
				//t.Ropen(strings.NewReader(fmt.Sprint(v)), nil)
                log.Printf("running mkdir 2")
				t.Ropen(mkdir(v), nil)
			}
		case styx.Tstat:
			t.Rstat(fi, nil)
		case styx.Tcreate:
			switch v := file.(type) {
			case map[string]interface{}:
				if t.Mode.IsDir() {
					dir := make(map[string]interface{})
					v[t.Name] = dir
					t.Rcreate(mkdir(dir), nil)
				} else {
					v[t.Name] = new(bytes.Buffer)
					t.Rcreate(&fakefile{
						v:   v[t.Name],
						set: func(s string) { v[t.Name] = s },
					}, nil)
				}
			case []interface{}:
				i, err := strconv.Atoi(t.Name)
				if err != nil {
					t.Rerror("member of an array must be a number: %s", err)
					break
				}
				if t.Mode.IsDir() {
					dir := make(map[string]interface{})
					v[i] = dir
					t.Rcreate(mkdir(dir), nil)
				} else {
					v[i] = new(bytes.Buffer)
					t.Rcreate(&fakefile{
						v:   v[i],
						set: func(s string) { v[i] = s },
					}, nil)
				}
			default:
				t.Rerror("%s is not a directory", t.Path())
			}
		case styx.Tremove:
			switch v := file.(type) {
			case map[string]interface{}:
				if len(v) > 0 {
					t.Rerror("directory is not empty")
					break
				}
				if parent != nil {
					if m, ok := parent.(map[string]interface{}); ok {
						delete(m, path.Base(t.Path()))
						t.Rremove(nil)
					} else {
						t.Rerror("cannot delete array element yet")
						break
					}
				} else {
					t.Rerror("permission denied")
				}
			default:
				if parent != nil {
					if m, ok := parent.(map[string]interface{}); ok {
						delete(m, path.Base(t.Path()))
						t.Rremove(nil)
					} else {
						t.Rerror("cannot delete array element")
					}
				} else {
					t.Rerror("permission denied")
				}
			}
		}
	}
}
