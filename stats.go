package datadog

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/garukun/datadog-api"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"google.golang.org/api/support/bundler"
)

// statsExporter exports stats to Datadog monitoring.
type statsExporter struct {
	bundler *bundler.Bundler
	o       Options

	c *datadog.Client
}

func newStatsExporter(o Options) (*statsExporter, error) {
	if o.APIKey == "" {
		return nil, fmt.Errorf("missing Datadog API key")
	}

	httpClient := o.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	e := &statsExporter{
		o: o,
		c: &datadog.Client{
			APIKey:     o.APIKey,
			HTTPClient: httpClient,
		},
	}

	e.bundler = bundler.NewBundler((*view.Data)(nil), func(bundle interface{}) {
		vds := bundle.([]*view.Data)
		e.handleUpload(vds...)
	})
	if e.o.BundleDelayThreshold > 0 {
		e.bundler.DelayThreshold = e.o.BundleDelayThreshold
	}
	if e.o.BundleCountThreshold > 0 {
		e.bundler.BundleCountThreshold = e.o.BundleCountThreshold
	}
	return e, nil
}

// ExportView exports to DataDog monitoring if view data has one or more rows.
func (e *statsExporter) ExportView(vd *view.Data) {
	if len(vd.Rows) == 0 {
		return
	}

	err := e.bundler.Add(vd, 1)
	switch err {
	case nil:
		return
	case bundler.ErrOversizedItem:
		go e.handleUpload(vd)
	case bundler.ErrOverflow:
		e.o.handleError(errors.New("failed to upload: buffer full"))
	default:
		e.o.handleError(err)
	}
}

// handleUpload handles uploading a slice of view.Data, as well as error handling.
func (e *statsExporter) handleUpload(vds ...*view.Data) {
	if err := e.uploadStats(vds); err != nil {
		e.o.handleError(err)
	}
}

// Flush waits for exported view data to be uploaded.
//
// This is useful if your program is ending and you do not
// want to lose recent spans.
func (e *statsExporter) Flush() {
	e.bundler.Flush()
}

func (e *statsExporter) uploadStats(vds []*view.Data) error {
	ts := newTimeSeriesRequest(vds)
	return e.c.UploadTimeSeries(ts)
}

func newTimeSeriesRequest(vds []*view.Data) datadog.TimeSeriesRequest {
	var tsReq datadog.TimeSeriesRequest
	hostname, _ := os.Hostname()

	for _, vd := range vds {
		name := normalizedTimeSeriesName(vd.View.Name)
		log.Print(name)

		for _, row := range vd.Rows {
			ts := datadog.TimeSeries{
				Name: name,
				Tags: newTags(row.Tags),
				Host: hostname,
			}

			addDataPoints(&ts, row, vd.End)
			tsReq.Series = append(tsReq.Series, ts)
		}
	}

	return tsReq
}

// normalizedTimeSeriesName normalizes invalid characters per Datadog spec from the given view name.
func normalizedTimeSeriesName(viewName string) string {
	return strings.Replace(viewName, "/", ".", -1)
}

func newTags(tags []tag.Tag) []string {
	ddTags := make([]string, 0, len(tags))

	for _, t := range tags {
		ddTags = append(ddTags, fmt.Sprintf("%s:%s", t.Key.Name(), t.Value))
	}

	return ddTags
}

func addDataPoints(ts *datadog.TimeSeries, r *view.Row, timestamp time.Time) {
	switch data := r.Data.(type) {
	case *view.CountData:
		ts.Points = append(ts.Points, datadog.DataPoint{
			Timestamp: timestamp.Unix(),
			Value:     float64(data.Value),
		})
	case *view.SumData:
		ts.Points = append(ts.Points, datadog.DataPoint{
			Timestamp: timestamp.Unix(),
			Value:     data.Value,
		})
	case *view.LastValueData:
		ts.Points = append(ts.Points, datadog.DataPoint{
			Timestamp: timestamp.Unix(),
			Value:     data.Value,
		})
	}
}
