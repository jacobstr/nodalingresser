# NodalIngresser

NodalIngresser updates an A record for you with the IP addresses of nodes in
your GKE cluster. This is a somewhat silly name for a thing that makes GKE
worker nodes more easily accessible on the public internet without incurring
the $20/mo cost of fronting my apps with an ILB.

---

*Heavily inspired by* this [blog
post](https://www.doxsey.net/blog/kubernetes--the-surprisingly-affordable-platform-for-personal-projects). The gist of
his clever approach is:

* Minimal GKE clusters can be had for ~$5/month.
* But it costs $20/month for an ILB.
* So put ingress-nginx on your worker nodes, edit the firewall rules on them to
  allow https, and manage DNS records on the fly as your fungible nodes go in and out of existence.

> Note: this guy doesn't edit firewall rules. My personal GKE cluster just
> has 443 open on the worker node pool.

This isn't as robust as a proper load balancer with health checks to your
various backends. DNS is dumb, and with multiple nodes, your clients may find
themselves round-robining across unhealthy nodes that are being shut down.

The author of the orignal post was using [Cloudflares DNS](https://github.com/calebdoxsey/kubernetes-cloudflare-sync), and I'm using Google's CloudDNS for my purposes.

## Usage

```
usage: nodalingresser [<flags>]

Automatically updates Google CloudDNS A Records for GKE Nodes.

Flags:
  --help                   Show context-sensitive help (also try --help-long and --help-man).
  --kubeconfig=KUBECONFIG  Path to kubeconfig file. Leave unset to use in-cluster config.
  --debug                  Enable debug logging.
  --client-go-verbosity=CLIENT-GO-VERBOSITY
                           Set client go verbosity level.
  --google-dns-service-account=GOOGLE-DNS-SERVICE-ACCOUNT
                           Path to service account json file with CloudDNS permissions.
  --google-dns-project=GOOGLE-DNS-PROJECT
                           Name of the project to modify records in.
  --google-dns-zone=GOOGLE-DNS-ZONE
                           Name of the zone to modify records in.
  --google-dns-record=GOOGLE-DNS-RECORD
                           Name of the record to modify.
```
