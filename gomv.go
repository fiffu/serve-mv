package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
)

type Options struct {
	Directory string
	Domain    string
	Subdomain string
	Port      int
}

type SystemJSON struct {
	GameTitle string `json:"gameTitle"`
}

func MustNewSystemJSON(directory string) (ret SystemJSON) {
	jsonPath := path.Join(directory, "www", "data", "System.json")
	jsonBytes := mustReturn(os.ReadFile(jsonPath))
	must(json.Unmarshal(jsonBytes, &ret))
	return
}

type cachedResponse struct {
	status int
	header http.Header
	buf    bytes.Buffer
}

func (cr *cachedResponse) writeTo(w http.ResponseWriter) {
	for k, vs := range cr.header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(cr.status)
	w.Write(cr.buf.Bytes())
}

func (cr *cachedResponse) Header() http.Header {
	if cr.header == nil {
		cr.header = http.Header{}
	}
	return cr.header
}

func (cr *cachedResponse) Write(data []byte) (int, error) {
	return cr.buf.Write(data)
}

func (cr *cachedResponse) WriteHeader(statusCode int) {
	cr.status = statusCode
}

type gameSrv struct {
	port    int
	dir     string
	fileSrv http.Handler
	cache   sync.Map

	cacheHit, cacheMiss int64
}

func (gs *gameSrv) initialize(opts Options) {
	gs.port = opts.Port
	gs.dir = opts.Directory
	gs.fileSrv = http.FileServer(http.Dir(opts.Directory))
	gs.cache = sync.Map{}
	gs.cacheHit = 0
	gs.cacheMiss = 0
}

func (gs *gameSrv) titleHash() string {
	title := MustNewSystemJSON(gs.dir).GameTitle

	sum := md5.Sum([]byte(title))
	titleHash := hex.EncodeToString(sum[:])
	return titleHash
}

func (gs *gameSrv) PrintReport() {
	fmt.Println()
	if gs == nil {
		return
	}

	cacheTotal := gs.cacheHit + gs.cacheMiss
	cacheRatio := float64(gs.cacheHit) / float64(cacheTotal)
	if (cacheTotal) > 0 {
		fmt.Printf(
			"Cache hit rate: %d / %d = %.0f%%",
			gs.cacheHit,
			cacheTotal,
			cacheRatio*100,
		)
		fmt.Println()
	}
}

func (gs *gameSrv) Start() {
	http.Handle("/", gs)
	http.ListenAndServe(fmt.Sprintf(":%d", gs.port), nil)
}

func (gs *gameSrv) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	var resp *cachedResponse

	cached, ok := gs.cache.Load(path)
	if ok {
		resp = cached.(*cachedResponse)
		gs.cacheHit++
	} else {
		fmt.Println("[miss]", path)
		gs.cacheMiss++

		resp = gs.fetch(w, r)

		if resp.status == http.StatusNotFound {
			if altPath := gs.glob(path); altPath != "" {
				r.URL.Path = altPath

				resp = gs.fetch(w, r)
			}
		}

		gs.cache.Store(path, resp)
	}

	if resp.status == http.StatusNotFound {
		fmt.Println("[404 ]", path)
	}
	resp.writeTo(w)
}

func (gs *gameSrv) fetch(w http.ResponseWriter, r *http.Request) *cachedResponse {
	ret := &cachedResponse{}
	gs.fileSrv.ServeHTTP(ret, r)
	return ret
}

func (gs *gameSrv) glob(path string) string {
	root := gs.dir
	parent := filepath.Dir(path)
	file := filepath.Base(path)

	globStr := fmt.Sprintf("%s/%s/%s", root, parent, CaseInsensitiveGlobstr(file))

	matches, _ := filepath.Glob(globStr)
	if len(matches) == 1 {
		return matches[0]
	}
	return ""
}

var Server = &gameSrv{}

func Listen(opts Options) error {
	Server.initialize(opts)

	fmt.Printf("Hosting game on: http://gomv-%s.localtest.me:%d/www/index.html\n", Server.titleHash(), opts.Port)
	Server.Start()

	return nil
}
