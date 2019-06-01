package dummy

import (
	"go.uber.org/zap"

	"github.com/jacobstr/nodalingresser/internal/logger"
)

type dummyDNSUpdater struct {
	logger *zap.Logger
}

type Option func(o *dummyDNSUpdater)

func WithLogger(l *zap.Logger) Option {
	return func(o *dummyDNSUpdater) {
		o.logger = l
	}
}

func NewDummyDNSUpdater(opts ...Option) *dummyDNSUpdater {
	u := &dummyDNSUpdater{
		logger: logger.DefaultLogger,
	}
	for _, o := range opts {
		o(u)
	}
	return u
}

func (d *dummyDNSUpdater) Update(ips []string) error {
	d.logger.Debug("got update invocation", zap.Strings("ips", ips))
	return nil
}
