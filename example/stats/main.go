package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/garukun/opencensus-go-exporter-datadog"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Create measures. The program will record measures for the size of
// processed videos and the number of videos marked as spam.
var videoSize = stats.Int64("measure.datadog.preferred.name.convention.video_size", "size of processed videos", stats.UnitBytes)
var videoCount = stats.Int64("measure.datadog.preferred.name.convention.video_count", "number of processed videos", stats.UnitDimensionless)
var videoCountCum = stats.Int64("measure.datadog.preferred.name.convention.video_count_cum", "cumulative number of processed videos", stats.UnitDimensionless)

func main() {
	ctx := context.Background()

	exporter, err := datadog.NewExporter(datadog.Options{
		APIKey: "some-api-key",
	})
	if err != nil {
		log.Fatal(err)
	}
	view.RegisterExporter(exporter)

	// Set reporting period to report data at every second.
	view.SetReportingPeriod(1 * time.Second)

	tkLocalTesting, _ := tag.NewKey("local_testing")
	tkOpenCensus, _ := tag.NewKey("opencensus")
	tkAnotherOne, _ := tag.NewKey("anotherone")
	ctx, _ = tag.New(ctx, tag.Insert(tkLocalTesting, "macbook"), tag.Insert(tkOpenCensus, "datadog"))

	// Create view to see the processed video size cumulatively.
	// Subscribe will allow view data to be exported.
	// Once no longer need, you can unsubscribe from the view.
	if err := view.Register(
		ochttp.ClientLatencyView,
		ochttp.ClientResponseCountByStatusCode,
		&view.View{
			Name:        "view.datadog.preferred.name.convention.video_size",
			Description: "processed video size over time",
			Measure:     videoSize,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{tkLocalTesting, tkOpenCensus, tkAnotherOne},
		}, &view.View{
			Name:        "view.datadog.preferred.name.convention.video_count",
			Description: "processed video count over time",
			Measure:     videoCount,
			Aggregation: view.Count(),
			TagKeys:     []tag.Key{tkLocalTesting, tkAnotherOne},
		}, &view.View{
			Name:        "view.datadog.preferred.name.convention.video_count_cum",
			Description: "cumulatively processed video count over time",
			Measure:     videoCountCum,
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tkLocalTesting, tkOpenCensus, tkAnotherOne},
		},
	); err != nil {
		log.Fatalf("Cannot subscribe to the view: %v", err)
	}

	processVideo(ctx)

	log.Print("sleeping for a minute...")
	time.Sleep(3 * time.Minute)
	exporter.Flush()
	log.Print("all metrics flushed!")
}

func processVideo(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)

	instrumentedClient := &http.Client{
		Transport: &ochttp.Transport{},
	}

	go func() {
		defer cancel()
		ticker := time.NewTicker(time.Second)
		i := 0
		for {
			select {
			case <-ticker.C:
				resp, err := instrumentedClient.Get(fmt.Sprintf("https://httpstat.us/%d", i%200+400))
				if err == nil {
					resp.Body.Close()
				}
				stats.Record(ctx, videoSize.M(int64(i*10)), videoCount.M(1), videoCountCum.M(1))
				i++
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}
