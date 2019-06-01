package dns

type DNSUpdater interface {
	Update(externalIPs []string) error
}
