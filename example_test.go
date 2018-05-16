package datadog_test

import (
	"log"
	"net/http"

	"github.com/garukun/opencensus-go-exporter-datadog"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
)

func Example() {
	exporter, err := datadog.NewExporter(datadog.Options{APIKey: "datadog-api-key"})
	if err != nil {
		log.Fatal(err)
	}

	// Export to Datadog.
	view.RegisterExporter(exporter)

	// Subscribe views to see stats in Datadog.
	if err := view.Register(
		ochttp.ClientLatencyView,
		ochttp.ClientResponseCountByStatusCode,
	); err != nil {
		log.Fatal(err)
	}

	instrumentedClient := &http.Client{
		Transport: &ochttp.Transport{},
	}


	// Use the instrumented client so that the metrics are collected.
	resp, err := instrumentedClient.Get("https://opencensus.io")
	if err != nil {
		return
	}
	resp.Body.Close()

	resp, err = instrumentedClient.Get("https://datadog.com")
	if err != nil {
		return
	}
	resp.Body.Close()

	// Invoke Flush to ensure metrics are uploaded to Datadog.
	exporter.Flush()
}
