package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gopkg.in/alecthomas/kingpin.v2"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	// load gcp auth plugin for local development.
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"

	"github.com/jacobstr/nodalingresser/internal/controller"
	"github.com/jacobstr/nodalingresser/internal/dns"
	dns_dummy "github.com/jacobstr/nodalingresser/internal/dns/dummy"
	dns_google "github.com/jacobstr/nodalingresser/internal/dns/google"
)

func BuildConfigFromFlags(kubecfg string) (*rest.Config, error) {
	if kubecfg != "" {
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubecfg},
			&clientcmd.ConfigOverrides{},
		).ClientConfig()
	}
	return rest.InClusterConfig()
}

func main() {
	var (
		app                 = kingpin.New(filepath.Base(os.Args[0]), "Automatically updates Google CloudDNS A Records for GKE Nodes.").DefaultEnvars()
		kubecfg             = app.Flag("kubeconfig", "Path to kubeconfig file. Leave unset to use in-cluster config.").String()
		debug               = app.Flag("debug", "Enable debug logging.").Bool()
		client_go_verbosity = app.Flag("client-go-verbosity", "Set client go verbosity level.").Int()

		google_dns_service_account = app.Flag("google-dns-service-account", "Path to service account json file with CloudDNS permissions.").String()
		google_dns_project         = app.Flag("google-dns-project", "Name of the project to modify records in.").String()
		google_dns_zone            = app.Flag("google-dns-zone", "Name of the zone to modify records in.").String()
		google_dns_record          = app.Flag("google-dns-record", "Name of the record to modify.").String()
	)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	zcfg := zap.NewProductionConfig()
	if *debug {
		zcfg = zap.NewDevelopmentConfig()
	}

	glogWorkaround(*client_go_verbosity)

	logger, err := zcfg.Build()
	defer logger.Sync()

	kingpin.FatalIfError(err, "cannot instantiate logger")

	cfg, err := BuildConfigFromFlags(*kubecfg)
	kingpin.FatalIfError(err, "cannot create Kubernetes client configuration")

	cs, err := kubernetes.NewForConfig(cfg)
	kingpin.FatalIfError(err, "cannot create Kubernetes client")

	factory := informers.NewSharedInformerFactory(cs, time.Minute)
	lister := factory.Core().V1().Nodes().Lister()

	var updater dns.DNSUpdater
	if *google_dns_service_account != "" {
		logger.Debug("using Google CloudDNS")
		updater, err = withGoogleUpdater(
			*google_dns_service_account,
			*google_dns_project,
			*google_dns_zone,
			*google_dns_record,
			logger,
		)
		kingpin.FatalIfError(err, "could not enable CloudDNS")
	} else {
		updater = dns_dummy.NewDummyDNSUpdater(dns_dummy.WithLogger(logger))
	}

	syncher := controller.NewDNSSyncher(lister, updater, controller.WithLogger(logger))
	informer := factory.Core().V1().Nodes().Informer()

	informer.AddEventHandler(syncher)

	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error {
		informer.Run(ctx.Done())
		return nil
	})
	logger.Debug("nodalingresser is running...")
	kingpin.FatalIfError(g.Wait(), "failed")
}

func withGoogleUpdater(sa string, project string, zone string, record string, logger *zap.Logger) (dns.DNSUpdater, error) {
	if sa == "" {
		return nil, errors.New("a google-dns-service-account is required when using google DNS")
	}

	if project == "" {
		return nil, errors.New("a google-dns-project is required when using google DNS")
	}

	if zone == "" {
		return nil, errors.New("a google-dns-zone is required when using google DNS")
	}

	if record == "" {
		return nil, errors.New("a google-dns-record is required when using google DNS")
	}

	return dns_google.NewCloudDNSUpdater(sa, project, zone, record, dns_google.WithLogger(logger))
}

// Many Kubernetes client things depend on glog. glog gets sad when flag.Parse()
// is not called before it tries to emit a log line. flag.Parse() fights with
// kingpin.
func glogWorkaround(level int) {
	lvl := fmt.Sprintf("-v=%d", level)
	klog.InitFlags(nil)
	os.Args = []string{os.Args[0], lvl, "-vmodule="}
	flag.Parse()
}
