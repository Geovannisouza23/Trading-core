package nats

import (
	"context"
	"fmt"
	"strings"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Connect dials url and ensures the shared `{prefix}_EVENTS` JetStream
// stream exists (subject filter `{prefix}.>`), matching quant-engine's
// own stream config exactly (src/infrastructure/messaging/nats/jetstream.rs)
// — whichever side starts first creates it; CreateOrUpdateStream is
// idempotent, so there is no ordering requirement between the two
// services.
func Connect(ctx context.Context, url, streamPrefix string) (*natsgo.Conn, jetstream.JetStream, error) {
	conn, err := natsgo.Connect(url, natsgo.Timeout(5*time.Second), natsgo.MaxReconnects(-1))
	if err != nil {
		return nil, nil, fmt.Errorf("connecting to nats at %s: %w", url, err)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("building jetstream context: %w", err)
	}

	// Stream name is uppercased (matching quant-engine's
	// format!("{}_EVENTS", prefix.to_uppercase()) exactly) — subjects
	// stay lowercase. Getting this wrong doesn't fail loudly: JetStream
	// rejects the second stream with "subjects overlap with an existing
	// stream" rather than silently creating a second one, but only once
	// both sides actually try to provision it against the same broker —
	// verified empirically via the two docker-compose stacks together.
	streamName := strings.ToUpper(streamPrefix) + "_EVENTS"
	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      streamName,
		Subjects:  []string{streamPrefix + ".>"},
		Retention: jetstream.LimitsPolicy,
		MaxAge:    30 * 24 * time.Hour,
	})
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("ensuring jetstream stream %s: %w", streamName, err)
	}

	return conn, js, nil
}
