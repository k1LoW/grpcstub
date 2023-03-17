package grpcstub

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

type config struct {
	protos            []string
	importPaths       []string
	useTLS            bool
	cacert, cert, key []byte
}

type Option func(*config) error

func Proto(proto string) Option {
	return func(c *config) error {
		protos := []string{}
		if f, err := os.Stat(proto); err == nil {
			if !f.IsDir() {
				c.protos = unique(append(c.protos, proto))
				return nil
			}
			proto = filepath.Join(proto, "*")
		}
		base, pattern := doublestar.SplitPattern(filepath.ToSlash(proto))
		c.importPaths = unique(append(c.importPaths, base))
		abs, err := filepath.Abs(base)
		if err != nil {
			return err
		}
		fsys := os.DirFS(abs)
		if err := doublestar.GlobWalk(fsys, pattern, func(p string, d fs.DirEntry) error {
			if d.IsDir() {
				return nil
			}
			protos = unique(append(protos, filepath.Join(base, p)))
			return nil
		}); err != nil {
			return err
		}
		if len(protos) == 0 {
			c.protos = unique(append(c.protos, proto))
		} else {
			c.protos = unique(append(c.protos, protos...))
		}
		return nil
	}
}

func Protos(protos []string) Option {
	return func(c *config) error {
		for _, p := range protos {
			opt := Proto(p)
			if err := opt(c); err != nil {
				return err
			}
		}
		return nil
	}
}

func ImportPaths(paths []string) Option {
	return func(c *config) error {
		c.importPaths = unique(append(c.importPaths, paths...))
		return nil
	}
}

func UseTLS(cacert, cert, key []byte) Option {
	return func(c *config) error {
		c.useTLS = true
		c.cacert = cacert
		c.cert = cert
		c.key = key
		return nil
	}
}

func unique(in []string) []string {
	u := []string{}
	m := map[string]struct{}{}
	for _, s := range in {
		if _, ok := m[s]; ok {
			continue
		}
		u = append(u, s)
		m[s] = struct{}{}
	}
	return u
}
