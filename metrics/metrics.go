package metrics

//All credits to https://github.com/eBay/fabio/tree/master/metrics
import (
	"errors"
	"fmt"
	logger "log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cyberdelia/go-metrics-graphite"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

var pfx string

func Clear() {
	log.Info("Unregistering all metrics.")
	metrics.DefaultRegistry.UnregisterAll()
}

func Mark(name string) {
	meter := metrics.GetOrRegisterMeter(name, metrics.DefaultRegistry)
	meter.Mark(1)
}

func Time(name string, function func()) {
	timer := metrics.GetOrRegisterTimer(name, metrics.DefaultRegistry)
	timer.Time(function)
}

func UpdateGauge(name string, value int64) {
	gauge := metrics.GetOrRegisterGauge(name, metrics.DefaultRegistry)
	gauge.Update(value)
}

func Init(cfg Config) error {
	pfx = cfg.Prefix
	if pfx == "default" {
		prefix, err := defaultPrefix()
		if err != nil {
			return err
		}
		pfx = prefix
	}

	collectSystemMetrics()

	switch cfg.Target {
	case "stdout":
		log.Info("Sending metrics to stdout")
		return initStdout(cfg.Interval.Duration)
	case "graphite":
		if cfg.Addr == "" {
			return errors.New("metrics: graphite addr missing")
		}

		log.Infof("Sending metrics to Graphite on %s as %q", cfg.Addr, pfx)
		return initGraphite(cfg.Addr, cfg.Interval.Duration)
	case "":
		log.Infof("Metrics disabled")
		return nil
	default:
		return fmt.Errorf("Invalid metrics target %s", cfg.Target)
	}
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

func defaultPrefix() (string, error) {
	host, err := hostname()
	if err != nil {
		log.WithError(err).Error("Problem with detecting prefix")
		return "", err
	}
	exe := filepath.Base(os.Args[0])
	return clean(host) + "." + clean(exe), nil
}

func initStdout(interval time.Duration) error {
	logger := logger.New(os.Stderr, "localhost: ", logger.Lmicroseconds)
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
