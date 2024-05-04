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
	healthCheck       bool
	disableReflection bool
	bufDirs           []string
	bufLocks          []string
	bufConfigs        []string
	bufModules        []string
}

type Option func(*config) error

// Proto append protos
func Proto(protos ...string) Option {
	return func(c *config) error {
		for _, p := range protos {
			opt := proto(p)
			if err := opt(c); err != nil {
				return err
			}
		}
		return nil
	}
}

// ImportPath set import paths
func ImportPath(paths ...string) Option {
	return func(c *config) error {
		c.importPaths = unique(append(c.importPaths, paths...))
		return nil
	}
}

// UseTLS enable TLS
func UseTLS(cacert, cert, key []byte) Option {
	return func(c *config) error {
		c.useTLS = true
		c.cacert = cacert
		c.cert = cert
		c.key = key
		return nil
	}
}

// EnableHealthCheck enable grpc.health.v1
func EnableHealthCheck() Option {
	return func(c *config) error {
		c.healthCheck = true
		return nil
	}
}

// DisableReflection disable Server Reflection Protocol
func DisableReflection() Option {
	return func(c *config) error {
		c.disableReflection = true
		return nil
	}
}

// BufDir use buf directory.
func BufDir(dirs ...string) Option {
	return func(c *config) error {
		c.bufDirs = unique(append(c.bufDirs, dirs...))
		return nil
	}
}

// BufLock use buf.lock for BSR.
func BufLock(locks ...string) Option {
	return func(c *config) error {
		c.bufLocks = unique(append(c.bufLocks, locks...))
		return nil
	}
}

// BufConfig use buf.yaml for BSR.
func BufConfig(configs ...string) Option {
	return func(c *config) error {
		c.bufConfigs = unique(append(c.bufConfigs, configs...))
		return nil
	}
}

// BufModule use buf modules for BSR.
func BufModule(module string) Option {
	return func(c *config) error {
		c.bufModules = unique(append(c.bufModules, module))
		return nil
	}
}

// BufModules use buf modules for BSR.
func BufModules(modules []string) Option {
	return func(c *config) error {
		for _, m := range modules {
			opt := BufModule(m)
			if err := opt(c); err != nil {
				return err
			}
		}
		return nil
	}
}

func proto(proto string) Option {
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
