package google

import (
	"context"

	"github.com/jacobstr/nodalingresser/internal/logger"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	v1 "google.golang.org/api/dns/v1"
	"google.golang.org/api/option"
)

var (
	defaultTTL int64 = 60
)

type Option func(o *cloudDNSUpdater)

func WithLogger(l *zap.Logger) Option {
	return func(o *cloudDNSUpdater) {
		o.logger = l
	}
}

type cloudDNSUpdater struct {
	project string
	zone    string
	record  string
	service *v1.Service
	logger  *zap.Logger
}

func NewCloudDNSUpdater(serviceAccountPath string, project string, zone string, record string, opts ...Option) (*cloudDNSUpdater, error) {
	dnsService, err := v1.NewService(context.TODO(), option.WithCredentialsFile(serviceAccountPath))
	if err != nil {
		return nil, errors.Wrap(err, "could not create cloudDNSUpdater")
	}

	l := &cloudDNSUpdater{
		logger:  logger.DefaultLogger,
		project: project,
		zone:    zone,
		record:  record,
		service: dnsService,
	}

	for _, o := range opts {
		o(l)
	}

	return l, nil
}

func getChange(name string, current *v1.ResourceRecordSet, desired []string) *v1.Change {
	deletions := []*v1.ResourceRecordSet(nil)
	if current != nil {
		deletions = append(deletions, current)
	}

	return &v1.Change{
		Additions: []*v1.ResourceRecordSet{
			&v1.ResourceRecordSet{
				Name:    name,
				Type:    "A",
				Rrdatas: desired,
				Ttl:     defaultTTL,
			},
		},
		Deletions: deletions,
	}
}

// Update updates our A records via CloudDNS. See: https://cloud.google.com/dns/docs/reference/v1/changes/create#examples
func (d *cloudDNSUpdater) Update(ips []string) error {
	d.logger.Debug(
		"got update invocation",
		zap.Strings("ips", ips),
		zap.String("project", d.project),
		zap.String("zone", d.zone),
		zap.String("record", d.record),
	)
	var current *v1.ResourceRecordSet

	// List existing A records for our zone, for a given record name.
	req := d.service.ResourceRecordSets.List(d.project, d.zone).Name(d.record).Type("A")
	if err := req.Pages(context.TODO(), func(page *v1.ResourceRecordSetsListResponse) error {
		for _, rrs := range page.Rrsets {
			d.logger.Debug(
				"saw Cloud DNS record",
				zap.Strings("rrdatas", rrs.Rrdatas),
				zap.String("name", rrs.Name),
				zap.String("type", rrs.Type),
			)
			if rrs.Name == d.record {
				d.logger.Info(
					"found existing record for update",
					zap.Strings("rrdatas", rrs.Rrdatas),
					zap.String("name", rrs.Name),
					zap.String("type", rrs.Type),
				)
				current = rrs
				return nil
			}
		}
		return nil
	}); err != nil {
		d.logger.Error("could not get existing DNS records", zap.Error(err))
		return errors.Wrap(err, "could not list existing DNS records")
	}

	rb := getChange(d.record, current, ips)
	resp, err := d.service.Changes.Create(d.project, d.zone, rb).Context(context.TODO()).Do()
	if err != nil {
		return errors.Wrap(err, "could not update DNS records")
	}

	d.logger.Info("updated Cloud DNS record ", zap.Int("status", resp.HTTPStatusCode))

	return nil
}
