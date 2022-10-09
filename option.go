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
		c.protos = append(c.protos, proto)
		return nil
	}
}

func Protos(protos []string) Option {
	return func(c *config) error {
		c.protos = append(c.protos, protos...)
		return nil
	}
}

func ImportPaths(paths []string) Option {
	return func(c *config) error {
		c.importPaths = append(c.importPaths, paths...)
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
