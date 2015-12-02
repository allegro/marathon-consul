package metrics

//All credits to https://github.com/eBay/fabio/tree/master/metrics
import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cyberdelia/go-metrics-graphite"
	"github.com/rcrowley/go-metrics"
)

var pfx string

func Init(cfg Config) error {
	if err := initMetrics(cfg); err != nil {
		return err
	}
	return nil
}

func Mark(name string) {
	meter := metrics.GetOrRegisterMeter(name, metrics.DefaultRegistry)
	meter.Mark(1)
}

func Time(name string, function func()) {
	timer := metrics.GetOrRegisterTimer(name, metrics.DefaultRegistry)
	timer.Time(function)
}

func initMetrics(cfg Config) error {
	pfx = cfg.Prefix
	if pfx == "default" {
		pfx = defaultPrefix()
	}

	switch cfg.Target {
	case "stdout":
		log.Printf("[INFO] Sending metrics to stdout")
		return initStdout(cfg.Interval)
	case "graphite":
		if cfg.Addr == "" {
			return errors.New("metrics: graphite addr missing")
		}

		log.Printf("[INFO] Sending metrics to Graphite on %s as %q", cfg.Addr, pfx)
		return initGraphite(cfg.Addr, cfg.Interval)
	case "":
		log.Printf("[INFO] Metrics disabled")
	default:
		log.Fatal("[FATAL] Invalid metrics target ", cfg.Target)
	}
	return nil
}

func TargetName(service, host, path string, targetURL *url.URL) string {
	return strings.Join([]string{
		clean(service),
		clean(host),
		clean(path),
		clean(targetURL.Host),
	}, ".")
}

func clean(s string) string {
	if s == "" {
		return "_"
	}
	s = strings.Replace(s, ".", "_", -1)
	s = strings.Replace(s, ":", "_", -1)
	return strings.ToLower(s)
}

// stubbed out for testing
var hostname = os.Hostname

func defaultPrefix() string {
	host, err := hostname()
	if err != nil {
		log.Fatal("[FATAL] ", err)
	}
	exe := filepath.Base(os.Args[0])
	return clean(host) + "." + clean(exe)
}

func initStdout(interval time.Duration) error {
	logger := log.New(os.Stderr, "localhost: ", log.Lmicroseconds)
	go metrics.Log(metrics.DefaultRegistry, interval, logger)
	return nil
}

func initGraphite(addr string, interval time.Duration) error {
	a, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return fmt.Errorf("metrics: cannot connect to Graphite: %s", err)
	}

	go graphite.Graphite(metrics.DefaultRegistry, interval, pfx, a)
	return nil
}
