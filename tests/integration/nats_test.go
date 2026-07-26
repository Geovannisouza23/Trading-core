//go:build integration

// This file proves trading-core's NATS publisher writes exactly what
// quant-engine's own consumers already hard-code — a real `nats:2.10-alpine`
// container (same image family quant-engine's docker-compose.yml uses),
// no quant-engine binary involved. The cross-repo end-to-end proof (Go
// publishes, the real quant-engine process consumes) lives in
// quant_grpc_test.go's heavier Docker-build setup and is a separate,
// more expensive test to run.
package integration_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"trading-core/internal/application/ports/output"
	natspkg "trading-core/internal/infrastructure/external/messaging/nats"
)

func newNatsContainer(t *testing.T) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        "nats:2.10-alpine",
		Cmd:          []string{"-js"},
		ExposedPorts: []string{"4222/tcp"},
		WaitingFor:   wait.ForListeningPort("4222/tcp").WithStartupTimeout(30 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("starting nats container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminating nats container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "4222/tcp")
	if err != nil {
		t.Fatalf("container mapped port: %v", err)
	}
	return "nats://" + host + ":" + port.Port()
}

// TestNatsPublisherWritesTheTwoFixedSubjectsQuantEngineExpects publishes
// through the real client/publisher wiring (client.Connect + NewPublisher,
// the exact code path providers.go uses) and asserts, via a plain core-NATS
// subscription (JetStream ingestion fans out to core subscribers on the
// same subject same as any other publish), that the envelope on the wire
// has the shape and subject quant-engine's decision_outcome_received/
// model_approved consumers already hard-code.
func TestNatsPublisherWritesTheTwoFixedSubjectsQuantEngineExpects(t *testing.T) {
	url := newNatsContainer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, js, err := natspkg.Connect(ctx, url, "quant")
	if err != nil {
		t.Fatalf("connecting to nats: %v", err)
	}
	t.Cleanup(func() { _ = conn.Drain() })

	decisionSub, err := conn.SubscribeSync("quant.decision.outcome.received")
	if err != nil {
		t.Fatalf("subscribing to decision outcome subject: %v", err)
	}
	modelSub, err := conn.SubscribeSync("quant.model.approved")
	if err != nil {
		t.Fatalf("subscribing to model approved subject: %v", err)
	}
	if err := conn.Flush(); err != nil {
		t.Fatalf("flushing subscriptions: %v", err)
	}

	publisher := natspkg.NewPublisher(js)

	if err := publisher.PublishDecisionOutcome(ctx, output.RegisterDecisionOutcomeInput{
		DecisionID:   "decision-abc",
		OrderCreated: true,
		Executed:     true,
		EntryPrice:   "50000.00",
		ExitPrice:    "50500.00",
		Quantity:     "0.01",
		Pnl:          "5.00",
	}); err != nil {
		t.Fatalf("PublishDecisionOutcome: %v", err)
	}

	if err := publisher.PublishModelApproved(ctx, output.ReloadApprovedModelInput{
		ModelName: "primary",
		Stage:     "CANARY",
	}); err != nil {
		t.Fatalf("PublishModelApproved: %v", err)
	}

	decisionMsg, err := decisionSub.NextMsg(5 * time.Second)
	if err != nil {
		t.Fatalf("waiting for decision outcome message: %v", err)
	}
	if decisionMsg.Subject != "quant.decision.outcome.received" {
		t.Errorf("subject = %q, want %q", decisionMsg.Subject, "quant.decision.outcome.received")
	}
	var decisionEnvelope natspkg.Envelope
	if err := json.Unmarshal(decisionMsg.Data, &decisionEnvelope); err != nil {
		t.Fatalf("decoding decision outcome envelope: %v", err)
	}
	if decisionEnvelope.EventType != "DecisionOutcomeReceived" {
		t.Errorf("event_type = %q, want %q", decisionEnvelope.EventType, "DecisionOutcomeReceived")
	}
	if decisionEnvelope.IdempotencyKey != "decision-outcome-decision-abc" {
		t.Errorf("idempotency_key = %q, want %q", decisionEnvelope.IdempotencyKey, "decision-outcome-decision-abc")
	}
	payload, ok := decisionEnvelope.Payload.(map[string]any)
	if !ok {
		t.Fatalf("payload type = %T, want a JSON object", decisionEnvelope.Payload)
	}
	if payload["decision_id"] != "decision-abc" {
		t.Errorf("payload.decision_id = %v, want %q", payload["decision_id"], "decision-abc")
	}

	modelMsg, err := modelSub.NextMsg(5 * time.Second)
	if err != nil {
		t.Fatalf("waiting for model approved message: %v", err)
	}
	if modelMsg.Subject != "quant.model.approved" {
		t.Errorf("subject = %q, want %q", modelMsg.Subject, "quant.model.approved")
	}
	var modelEnvelope natspkg.Envelope
	if err := json.Unmarshal(modelMsg.Data, &modelEnvelope); err != nil {
		t.Fatalf("decoding model approved envelope: %v", err)
	}
	if modelEnvelope.EventType != "ModelApproved" {
		t.Errorf("event_type = %q, want %q", modelEnvelope.EventType, "ModelApproved")
	}
	if modelEnvelope.IdempotencyKey != "model-approved-primary-CANARY" {
		t.Errorf("idempotency_key = %q, want %q", modelEnvelope.IdempotencyKey, "model-approved-primary-CANARY")
	}
}
