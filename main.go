package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ys3669/dns-track-expoter/config"
	"github.com/ys3669/dns-track-expoter/dns"
)

var (
	// DNS response time in seconds
	dnsResponseTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dns_response_time_seconds",
			Help: "DNS response time in seconds",
		},
		[]string{"fqdn", "record_type", "dns_server"},
	)

	// DNS resolution success/failure
	dnsResolutionSuccess = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dns_resolution_success",
			Help: "DNS resolution success (1 = success, 0 = failure)",
		},
		[]string{"fqdn", "record_type", "dns_server"},
	)

	// Number of resolved IP addresses
	dnsResolvedIpCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dns_resolved_ip_count",
			Help: "Number of IP addresses resolved for FQDN",
		},
		[]string{"fqdn", "record_type", "dns_server"},
	)

	// Total DNS query count
	dnsQueryTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dns_query_total",
			Help: "Total number of DNS queries performed",
		},
		[]string{"fqdn", "record_type", "dns_server", "status"},
	)

	// Resolved IP addresses (1 = IP exists for FQDN)
	dnsResolvedIpAddress = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dns_resolved_ip_address",
			Help: "Resolved IP addresses for FQDN (1 = IP exists)",
		},
		[]string{"fqdn", "record_type", "dns_server", "ip_address"},
	)
)

var (
	// Custom registry without Go runtime metrics
	customRegistry = prometheus.NewRegistry()
)

func init() {
	// Register metrics with custom registry (not default one)
	customRegistry.MustRegister(dnsResponseTime)
	customRegistry.MustRegister(dnsResolutionSuccess)
	customRegistry.MustRegister(dnsResolvedIpCount)
	customRegistry.MustRegister(dnsQueryTotal)
	customRegistry.MustRegister(dnsResolvedIpAddress)
}

func main() {
	// Parse command line flags
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting DNS trace exporter on port %d", cfg.Server.Port)
	log.Printf("Monitoring interval: %v", cfg.Monitoring.Interval)
	log.Printf("DNS timeout: %v", cfg.Monitoring.Timeout)

	// Create DNS resolver
	resolver := dns.NewResolver(
		dnsResponseTime,
		dnsResolutionSuccess,
		dnsResolvedIpCount,
		dnsQueryTotal,
		dnsResolvedIpAddress,
	)

	// Start DNS monitoring
	go func() {
		ticker := time.NewTicker(cfg.Monitoring.Interval)
		defer ticker.Stop()

		for {
			for _, target := range cfg.Targets {
				for _, dnsServer := range cfg.DNSServers {
					for _, recordType := range target.RecordTypes {
						log.Printf("Resolving %s (%s) via %s (%s)", target.FQDN, recordType, dnsServer.Name, dnsServer.Address)
						resolver.Lookup(target.FQDN, dnsServer.Address, recordType, cfg.Monitoring.Timeout)
					}
				}
			}
			<-ticker.C
		}
	}()

	// Setup HTTP server with custom registry
	http.Handle("/metrics", promhttp.HandlerFor(customRegistry, promhttp.HandlerOpts{}))

	listenAddr := cfg.GetListenAddress()
	log.Printf("Server starting on %s", listenAddr)

	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
