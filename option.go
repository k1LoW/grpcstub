package grpcstub

type config struct {
	protos            []string
	importPaths       []string
	useTLS            bool
	cacert, cert, key []byte
}

type Option func(*config) error

func Proto(proto string) Option {
	return func(c *config) error {
		c.protos = unique(append(c.protos, proto))
		return nil
	}
}

func Protos(protos []string) Option {
	return func(c *config) error {
		c.protos = unique(append(c.protos, protos...))
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
