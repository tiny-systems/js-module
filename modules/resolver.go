package modules

import (
	"context"
	"fmt"
	"github.com/grafana/sobek"
	"github.com/grafana/sobek/parser"
	"github.com/tiny-systems/js-module/lib"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Resolver struct {
	mu sync.Mutex
	// cache module-name => ModuleRecord
	cache map[string]cacheElement
	// built in modules
	goModules map[string]lib.Module
	//
	// virtual local file system
	fs fs.FS
}

type cacheElement struct {
	m   sobek.ModuleRecord
	err error
}

func NewResolver(fs fs.FS, goModules map[string]lib.Module, vu lib.VU) *Resolver {

	rt := vu.Runtime()
	// TODO:figure out if we can remove this
	_ = rt.GlobalObject().DefineDataProperty("vubox",
		rt.ToValue(vubox{vu: vu}), sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)

	return &Resolver{fs: fs, cache: make(map[string]cacheElement), goModules: goModules}
}

func (s *Resolver) Resolve(_ interface{}, specifier string) (sobek.ModuleRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k, ok := s.cache[specifier]
	if ok {
		return k.m, k.err
	}

	var (
		b   []byte
		err error
	)

	// check if module built int
	var modRecord sobek.ModuleRecord

	builtInModule, builtIn := s.goModules[specifier]
	if builtIn {
		modRecord = &goModule{
			m: builtInModule,
		}
	} else {

		if strings.HasPrefix(specifier, "http://") || strings.HasPrefix(specifier, "https://") {
			b, err = fetch(context.Background(), specifier)
		} else {
			b, err = fs.ReadFile(s.fs, specifier)
		}

		if err != nil {
			s.cache[specifier] = cacheElement{err: err}
			return nil, err
		}
		modRecord, err = sobek.ParseModule(specifier, string(b), s.Resolve, parser.WithSourceMapLoader(func(path string) ([]byte, error) {
			return fetch(context.Background(), path)
		}))
		if err != nil {
			s.cache[specifier] = cacheElement{err: err}
			return nil, err
		}
	}
	s.cache[specifier] = cacheElement{m: modRecord}

	return modRecord, nil
}

func fetch(ctx context.Context, u string) ([]byte, error) {

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		switch res.StatusCode {
		case http.StatusNotFound:
			return nil, fmt.Errorf("not found: %s", u)
		default:
			return nil, fmt.Errorf("wrong status code (%d) for: %s", res.StatusCode, u)
		}
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}
