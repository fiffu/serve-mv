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

type GameServer struct {
	port    int
	dir     string
	fileSrv http.Handler
	cache   map[string]*cachedResponse

	cacheHit, cacheMiss uint64
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

func NewGameServer(opts Options) *GameServer {
	return &GameServer{
		opts.Port,
		opts.Directory,
		http.FileServer(http.Dir(opts.Directory)),
		make(map[string]*cachedResponse),
		0, 0,
	}
}

func (gs *GameServer) TitleHash() string {
	title := MustNewSystemJSON(gs.dir).GameTitle

	sum := md5.Sum([]byte(title))
	titleHash := hex.EncodeToString(sum[:])
	return titleHash
}

func (gs *GameServer) Start() {
	http.Handle("/", gs)

	server := &http.Server{Addr: fmt.Sprintf(":%d", gs.port), Handler: nil}
	server.RegisterOnShutdown(func() {
		fmt.Println(
			"Cache hit rate: %d/%d (%.2f%)",
			gs.cacheHit,
			gs.cacheHit+gs.cacheMiss,
			gs.cacheHit/(gs.cacheHit+gs.cacheMiss)*100,
		)
		fmt.Println("Server stopping...")
	})
	server.ListenAndServe()
}

func (gs *GameServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	cached, ok := gs.cache[path]
	if ok {
		gs.cacheHit++
	} else {
		fmt.Println("[miss]", path)
		gs.cacheMiss++

		cached = gs.fetch(w, r)

		if cached.status == http.StatusNotFound {
			if altPath := gs.glob(path); altPath != "" {
				r.URL.Path = altPath
				cached = gs.fetch(w, r)
			}
		}

		gs.cache[path] = cached
	}

	if cached.status == http.StatusNotFound {
		fmt.Println("[404 ]", path)
	}
	cached.writeTo(w)
}

func (gs *GameServer) fetch(w http.ResponseWriter, r *http.Request) *cachedResponse {
	ret := &cachedResponse{}
	gs.fileSrv.ServeHTTP(ret, r)
	return ret
}

func (gs *GameServer) glob(path string) string {
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

func Listen(opts Options) error {
	gs := NewGameServer(opts)

	fmt.Printf("Hosting game on: http://gomv-%s.localtest.me:%d/www/index.html\n", gs.TitleHash(), opts.Port)
	gs.Start()

	return nil
}
