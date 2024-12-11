package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type Options struct {
	Directory string
	Domain    string
	Subdomain string
	Port      int
}

type TempDNS struct {
	Options
	SystemJSON
	HostsFilePath       string
	HostsFileBackupPath string
}

func NewTempDNS(opts Options) (*TempDNS, error) {
	sj, err := NewSystemJSON(opts.Directory)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().UnixMicro()
	return &TempDNS{
		Options:             opts,
		SystemJSON:          sj,
		HostsFilePath:       "/etc/hosts",
		HostsFileBackupPath: fmt.Sprintf("/etc/hosts.gomv.%d.bk", timestamp),
	}, nil
}

func (t *TempDNS) getDomain() string {
	if t.Options.Domain != DefaultDomain {
		return t.Options.Domain
	}
	return DefaultDomain
}

func (t *TempDNS) getSubdomain() string {
	if t.Options.Subdomain != DefaultSubdomain {
		return t.Options.Subdomain
	}
	return t.SystemJSON.GameTitle
}

func (t *TempDNS) HostName() string {
	return fmt.Sprintf("%s.%s", t.hashMD5(t.getSubdomain()), t.getDomain())
}

func (t *TempDNS) HostsRecord() string {
	return fmt.Sprintf("\n127.0.0.1\t%s\n", t.HostName())
}

func (t *TempDNS) hashMD5(str string) string {
	sum := md5.Sum([]byte(str))
	return hex.EncodeToString(sum[:])
}

func (t *TempDNS) WithDNS(callback func(hostName string)) (err error) {
	hf := HostsFileSwapper{
		tempRecord: t.HostsRecord(),
		path:       t.HostsFilePath,
		backupPath: t.HostsFileBackupPath,
	}
	err = hf.Setup()
	if err != nil {
		return err
	}

	defer func(hf HostsFileSwapper) {
		err = hf.TearDown()
	}(hf)

	callback(t.HostName())

	return nil
}

type HostsFileSwapper struct {
	tempRecord string // example: 127.0.0.1     gomv.local
	path       string // example: /etc/hosts
	backupPath string // example: /etc/hosts.gomv-1704458334000000.bk
}

func (hf HostsFileSwapper) Setup() error {
	fmt.Printf("Backing up %s to %s\n", hf.path, hf.backupPath)
	if err := hf.makeBackup(); err != nil {
		return err
	}
	fmt.Printf("Adding in %s: %s\n", hf.path, strings.TrimSpace(hf.tempRecord))
	return hf.appendToFile(hf.path, []byte(hf.tempRecord))
}

func (hf HostsFileSwapper) TearDown() error {
	src, err := os.Open(hf.path)
	if err != nil {
		return err
	}
	defer src.Close()

	srcBytes, err := io.ReadAll(src)
	if err != nil {
		return err
	}

	fmt.Printf("Removing in %s: %s\n", hf.path, strings.TrimSpace(hf.tempRecord))
	replacement := bytes.ReplaceAll(srcBytes, []byte(hf.tempRecord), []byte(""))
	if err := hf.replaceFile(hf.path, replacement); err != nil {
		return err
	}

	fmt.Printf("Removing backup %s\n", hf.backupPath)
	return os.Remove(hf.backupPath)
}

func (hf HostsFileSwapper) makeBackup() error {
	src, err := os.Open(hf.path)
	if err != nil {
		return err
	}
	defer src.Close()

	backup, err := os.Create(hf.backupPath)
	if err != nil {
		return err
	}
	defer backup.Close()

	if _, err := io.Copy(backup, src); err != nil {
		return err
	}

	return nil
}

func (hf HostsFileSwapper) appendToFile(path string, content []byte) error {
	return hf.editFile(path, content, false)
}

func (hf HostsFileSwapper) replaceFile(path string, content []byte) error {
	return hf.editFile(path, content, true)
}

// editFile edits an existing file.
func (hf HostsFileSwapper) editFile(path string, content []byte, truncate bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	flag := os.O_APPEND
	if truncate {
		flag = os.O_TRUNC
	}

	file, err := os.OpenFile(path, os.O_WRONLY|flag, info.Mode())
	if err != nil {
		return err
	}

	if _, err := file.Write(content); err != nil {
		return err
	}
	return file.Sync()
}

type SystemJSON struct {
	GameTitle string `json:"gameTitle"`
}

func NewSystemJSON(directory string) (ret SystemJSON, err error) {
	jsonPath := path.Join(directory, "www", "data", "System.json")
	var jsonBytes []byte
	if jsonBytes, err = os.ReadFile(jsonPath); err != nil {
		return
	}
	json.Unmarshal(jsonBytes, &ret)
	return
}

type GameServer struct {
	port    int
	fileSrv http.Handler
	dir     fs.FS
	cache   map[string]*cachedResponse
}

type cachedResponse struct {
	status int
	header http.Header
	buf    bytes.Buffer
}

func (cr *cachedResponse) writeTo(w http.ResponseWriter) error {
	for k, vs := range cr.header {
		w.Header()[k] = vs
	}
	w.WriteHeader(cr.status)
	_, err := w.Write(cr.buf.Bytes())
	return err
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
		http.FileServer(http.Dir(opts.Directory)),
		os.DirFS(opts.Directory),
		make(map[string]*cachedResponse),
	}
}

func (gs *GameServer) Listen() {
	http.Handle("/", gs)
	http.ListenAndServe(fmt.Sprintf(":%d", gs.port), nil)
}

func (gs *GameServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	cached, ok := gs.cache[path]
	if !ok {
		cached = gs.fetch(w, r)

		if cached.status == http.StatusNotFound {
			if altPath := gs.glob(path); altPath != "" {
				r.URL.Path = altPath
				cached = gs.fetch(w, r)
			}
		}

		gs.cache[path] = cached
	}

	if err := cached.writeTo(w); err != nil {
		fmt.Printf("errored while writing cached response for %s: %v\n", path, err)
		w.WriteHeader(500)
	}
}

func (gs *GameServer) fetch(w http.ResponseWriter, r *http.Request) *cachedResponse {
	ret := &cachedResponse{}
	gs.fileSrv.ServeHTTP(ret, r)
	return ret
}

func (gs *GameServer) glob(path string) string {
	matches, _ := filepath.Glob(CaseInsensitiveGlobstr(path))
	if len(matches) == 1 {
		return matches[0]
	}
	return ""
}

func (gs *GameServer) WaitUntilInterrupt() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
}

func Listen(opts Options) error {
	tmp, err := NewTempDNS(opts)
	if err != nil {
		return err
	}

	return tmp.WithDNS(func(hostName string) {
		fmt.Printf("Starting server - - - http://%s:%d/www/index.html\n", hostName, opts.Port)

		game := NewGameServer(opts)
		game.Listen()
		game.WaitUntilInterrupt()

		fmt.Println("Interrupted! Shutting down...")
	})
}
