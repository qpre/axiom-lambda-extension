package flusher

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/axiomhq/axiom-lambda-extension/version"

	"github.com/axiomhq/axiom-go/axiom"
)

// Axiom Config
var (
	axiomToken    = os.Getenv("AXIOM_TOKEN")
	axiomDataset  = os.Getenv("AXIOM_DATASET")
	batchSize     = 1000
	flushInterval = 1 * time.Second
)

type Axiom struct {
	client        *axiom.Client
	events        []axiom.Event
	lastFlushTime time.Time
}

func New() (*Axiom, error) {
	client, err := axiom.NewClient(
		axiom.SetAPITokenConfig(axiomToken),
		axiom.SetUserAgent(fmt.Sprintf("axiom-lambda-extension/%s", version.Get())),
		axiom.SetNoRetry(),
	)
	if err != nil {
		return nil, err
	}

	f := &Axiom{
		client: client,
		events: make([]axiom.Event, 0),
	}

	return f, nil
}

func (f *Axiom) ShouldFlush() bool {
	return len(f.events) > batchSize || f.lastFlushTime.IsZero() || time.Since(f.lastFlushTime) > flushInterval
}

func (f *Axiom) Queue(event axiom.Event) {
	f.events = append(f.events, event)
}

func (f *Axiom) Flush() {
	f.lastFlushTime = time.Now()
	if len(f.events) == 0 {
		return
	}

	res, err := f.client.IngestEvents(context.Background(), axiomDataset, f.events)
	if err != nil {
		log.Println(fmt.Errorf("failed to ingest events: %w", err))
		// allow this batch to be retried again
		return
	} else if res.Failed > 0 {
		log.Printf("%d failures during ingesting, %s", res.Failed, res.Failures[0].Error)
	}
	f.events = f.events[:0] // Clear the batch.
}
