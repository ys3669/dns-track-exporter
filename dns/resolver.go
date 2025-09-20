package dns

import (
	"context"
	"net"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Result represents DNS resolution result
type Result struct {
	FQDN        string
	RecordType  string
	DNSServer   string
	IPs         []net.IPAddr
	Duration    time.Duration
	Success     bool
	Error       error
}

// Resolver handles DNS resolution with metrics
type Resolver struct {
	responseTime      *prometheus.GaugeVec
	resolutionSuccess *prometheus.GaugeVec
	resolvedIpCount   *prometheus.GaugeVec
	queryTotal        *prometheus.CounterVec
	resolvedIpAddress *prometheus.GaugeVec
}

// NewResolver creates a new DNS resolver with metrics
func NewResolver(responseTime, resolutionSuccess, resolvedIpCount *prometheus.GaugeVec,
	queryTotal *prometheus.CounterVec, resolvedIpAddress *prometheus.GaugeVec) *Resolver {
	return &Resolver{
		responseTime:      responseTime,
		resolutionSuccess: resolutionSuccess,
		resolvedIpCount:   resolvedIpCount,
		queryTotal:        queryTotal,
		resolvedIpAddress: resolvedIpAddress,
	}
}

// Lookup performs DNS resolution and updates metrics
func (r *Resolver) Lookup(fqdn, dnsServer, recordType string, timeout time.Duration) *Result {
	start := time.Now()

	// Create resolver with custom DNS server if specified
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Second * 5,
			}
			if dnsServer != "" {
				return d.DialContext(ctx, network, dnsServer+":53")
			}
			return d.DialContext(ctx, network, address)
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var ips []net.IPAddr
	var err error

	switch recordType {
	case "A":
		// IPv4 only
		ipv4s, lookupErr := resolver.LookupIP(ctx, "ip4", fqdn)
		if lookupErr == nil {
			for _, ip := range ipv4s {
				ips = append(ips, net.IPAddr{IP: ip})
			}
		}
		err = lookupErr
	case "AAAA":
		// IPv6 only
		ipv6s, lookupErr := resolver.LookupIP(ctx, "ip6", fqdn)
		if lookupErr == nil {
			for _, ip := range ipv6s {
				ips = append(ips, net.IPAddr{IP: ip})
			}
		}
		err = lookupErr
	default:
		// Both IPv4 and IPv6
		ips, err = resolver.LookupIPAddr(ctx, fqdn)
	}

	duration := time.Since(start)

	result := &Result{
		FQDN:       fqdn,
		RecordType: recordType,
		DNSServer:  dnsServer,
		IPs:        ips,
		Duration:   duration,
		Success:    err == nil,
		Error:      err,
	}

	// Update metrics
	r.updateMetrics(result)

	return result
}

// updateMetrics updates Prometheus metrics based on DNS resolution result
func (r *Resolver) updateMetrics(result *Result) {
	labels := prometheus.Labels{
		"fqdn":        result.FQDN,
		"record_type": result.RecordType,
		"dns_server":  result.DNSServer,
	}

	// Update response time
	r.responseTime.With(labels).Set(result.Duration.Seconds())

	if !result.Success {
		// DNS resolution failed
		r.resolutionSuccess.With(labels).Set(0)
		r.queryTotal.With(prometheus.Labels{
			"fqdn":        result.FQDN,
			"record_type": result.RecordType,
			"dns_server":  result.DNSServer,
			"status":      "failure",
		}).Inc()
		return
	}

	// DNS resolution succeeded
	r.resolutionSuccess.With(labels).Set(1)
	r.resolvedIpCount.With(labels).Set(float64(len(result.IPs)))
	r.queryTotal.With(prometheus.Labels{
		"fqdn":        result.FQDN,
		"record_type": result.RecordType,
		"dns_server":  result.DNSServer,
		"status":      "success",
	}).Inc()

	// Set metrics for each resolved IP
	for _, ip := range result.IPs {
		ipLabels := prometheus.Labels{
			"fqdn":        result.FQDN,
			"record_type": result.RecordType,
			"dns_server":  result.DNSServer,
			"ip_address":  ip.IP.String(),
		}
		r.resolvedIpAddress.With(ipLabels).Set(1)
	}
}