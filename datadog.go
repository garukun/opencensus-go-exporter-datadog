package datadog

import (
	"go.opencensus.io/stats/view"
	"log"
	"net/http"
	"time"
)

// Options contains options for configuring the exporter.
type Options struct {
	// APIKey provides required access rights to Datadog’s API. The API key is usually obtained
	// through DataDog admin console.
	APIKey string

	// HTTPClient allows Datadog API client to send HTTP requests through an alternative HTTP client.
	// By default, http.DefaultClient is used.
	// Optional.
	HTTPClient *http.Client

	// OnError is the hook to be called when there is
	// an error uploading the stats.
	// If no custom hook is set, errors are logged.
	// Optional.
	OnError func(err error)

	// BundleDelayThreshold determines the max amount of time
	// the exporter can wait before uploading view data to
	// the backend.
	// Optional.
	BundleDelayThreshold time.Duration

	// BundleCountThreshold determines how many view data events
	// can be buffered before batch uploading them to the backend.
	// Optional.
	BundleCountThreshold int
}

// Exporter is a stats.Exporter implementation that uploads data to Datadog.
type Exporter struct {
	statsExporter *statsExporter
}

// NewExporter creates a new Exporter that implements stats.Exporter.
func NewExporter(o Options) (*Exporter, error) {
	se, err := newStatsExporter(o)
	if err != nil {
		return nil, err
	}
	return &Exporter{
		statsExporter: se,
	}, nil
}

// ExportView exports to Datadog Monitoring if view data has one or more rows.
func (e *Exporter) ExportView(vd *view.Data) {
	e.statsExporter.ExportView(vd)
}

// Flush waits for exported data to be uploaded.
//
// This is useful if your program is ending and you do not
// want to lose recent stats or spans.
func (e *Exporter) Flush() {
	e.statsExporter.Flush()
}

func (o Options) handleError(err error) {
	if o.OnError != nil {
		o.OnError(err)
		return
	}
	log.Printf("Error exporting to Datadog: %v", err)
}
