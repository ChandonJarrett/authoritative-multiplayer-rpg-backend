// Package observability contains dependency-light metrics and instrumentation helpers.
package observability

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const unknownService = "unknown"

// Metrics stores process-local counters in a Prometheus-friendly format.
type Metrics struct {
	service string

	mu       sync.Mutex
	counters map[string]uint64
	timers   map[string]timerStats
}

type timerStats struct {
	Count       uint64
	TotalMillis uint64
	MaxMillis   uint64
}

// NewMetrics creates an in-memory metrics collector.
func NewMetrics(service string) *Metrics {
	service = labelValue(service)
	if service == "" {
		service = unknownService
	}

	return &Metrics{
		service:  service,
		counters: make(map[string]uint64),
		timers:   make(map[string]timerStats),
	}
}

// Inc increments a named counter with labels.
func (m *Metrics) Inc(name string, labels map[string]string) {
	if m == nil {
		return
	}

	key := metricKey(name, withServiceLabel(m.service, labels))

	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[key]++
}

// ObserveDuration records a duration for a named metric with labels.
func (m *Metrics) ObserveDuration(name string, duration time.Duration, labels map[string]string) {
	if m == nil {
		return
	}

	millis := uint64(duration.Milliseconds())
	key := metricKey(name, withServiceLabel(m.service, labels))

	m.mu.Lock()
	defer m.mu.Unlock()

	stats := m.timers[key]
	stats.Count++
	stats.TotalMillis += millis
	if millis > stats.MaxMillis {
		stats.MaxMillis = millis
	}
	m.timers[key] = stats
}

// Handler returns an HTTP handler exposing metrics in Prometheus text format.
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if m == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("# metrics unavailable\n"))
			return
		}

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = w.Write([]byte(m.Render()))
	})
}

// Render returns all metrics in Prometheus text format.
func (m *Metrics) Render() string {
	if m == nil {
		return "# metrics unavailable\n"
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var b strings.Builder

	counterKeys := make([]string, 0, len(m.counters))
	for key := range m.counters {
		counterKeys = append(counterKeys, key)
	}
	sort.Strings(counterKeys)

	for _, key := range counterKeys {
		fmt.Fprintf(&b, "%s %d\n", key, m.counters[key])
	}

	timerKeys := make([]string, 0, len(m.timers))
	for key := range m.timers {
		timerKeys = append(timerKeys, key)
	}
	sort.Strings(timerKeys)

	for _, key := range timerKeys {
		stats := m.timers[key]
		fmt.Fprintf(&b, "%s_count %d\n", key, stats.Count)
		fmt.Fprintf(&b, "%s_total_ms %d\n", key, stats.TotalMillis)
		fmt.Fprintf(&b, "%s_max_ms %d\n", key, stats.MaxMillis)
	}

	return b.String()
}

func withServiceLabel(service string, labels map[string]string) map[string]string {
	out := make(map[string]string, len(labels)+1)
	out["service"] = service

	for key, value := range labels {
		out[labelName(key)] = labelValue(value)
	}

	return out
}

func metricKey(name string, labels map[string]string) string {
	name = metricName(name)
	if len(labels) == 0 {
		return name
	}

	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, labelName(key), labelValue(labels[key])))
	}

	return fmt.Sprintf("%s{%s}", name, strings.Join(parts, ","))
}

func metricName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown_metric"
	}

	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == ':':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}

	return b.String()
}

func labelName(value string) string {
	return metricName(value)
}

func labelValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	value = strings.ReplaceAll(value, "\n", `\n`)

	if value == "" {
		return "unknown"
	}

	return value
}
