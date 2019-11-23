package controller

import (
	"reflect"
	"sort"
	"sync"

	"github.com/jacobstr/nodalingresser/internal/dns"
	"github.com/jacobstr/nodalingresser/internal/logger"

	"go.uber.org/zap"

	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	listers_v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type Option func(o *dnsSyncher)

func WithLogger(l *zap.Logger) Option {
	return func(o *dnsSyncher) {
		o.logger = l
	}
}

type dnsSyncher struct {
	lister   listers_v1.NodeLister
	selector labels.Selector
	updater  dns.DNSUpdater

	logger *zap.Logger

	ips []string
	mux sync.Mutex
}

func NewDNSSyncher(lister listers_v1.NodeLister, updater dns.DNSUpdater, opts ...Option) cache.ResourceEventHandler {
	ds := &dnsSyncher{
		lister:   lister,
		selector: labels.NewSelector(),
		updater:  updater,
		logger:   logger.DefaultLogger,
	}

	for _, o := range opts {
		o(ds)
	}

	return ds
}

func (d *dnsSyncher) resync() {
	// We want to serialize resyncs of our internal cache of ips.
	d.mux.Lock()
	defer d.mux.Unlock()

	ips := []string(nil)
	nodes, err := d.lister.List(d.selector)
	if err != nil {
		return
	}

	for _, node := range nodes {
		for _, addr := range node.Status.Addresses {
			if addr.Type == core_v1.NodeExternalIP {
				ips = append(ips, addr.Address)
			}
		}
	}

	// Sort ips for consistent diffs when checking for updates.
	sort.Strings(ips)

	// We maintain a simplified representation of internal ips and check if they've
	// changed instead of responding to every update event. Nodes change state for
	// various reasons we may not care about.
	if reflect.DeepEqual(ips, d.ips) {
		d.logger.Debug("ip addresses unchanged")
		return
	}

	d.logger.Info("external ip addresses have changed, delegating to Updater to refresh records")
	if err := d.updater.Update(ips); err != nil {
		d.logger.Error("could not update DNS", zap.Error(err))
		return
	}

	d.ips = ips
}

func (d *dnsSyncher) OnAdd(o interface{}) {
	d.logger.Debug("OnAdd")
	d.resync()
}

func (d *dnsSyncher) OnDelete(o interface{}) {
	d.logger.Debug("OnDelete")
	d.resync()
}

func (d *dnsSyncher) OnUpdate(o interface{}, n interface{}) {
	d.logger.Debug("OnUpdate")
	d.resync()
}
