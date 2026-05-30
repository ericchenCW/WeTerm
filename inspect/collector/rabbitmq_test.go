package collector

import (
	"testing"

	"weops-inspect/model"
)

// fakeProbe constructs an rmqProbe with the stalled-queue thresholds wired up
// but no network state; tests exercise only the filter helpers.
func fakeProbe(ackMax, pubMin float64, blacklist ...string) *rmqProbe {
	bl := map[string]bool{}
	for _, v := range blacklist {
		bl[v] = true
	}
	return &rmqProbe{
		status:            &model.RabbitMQStatus{},
		backlogThreshold:  10000,
		stalledAckRateMax: ackMax,
		stalledPubRateMin: pubMin,
		stalledVHostBL:    bl,
	}
}

func TestIsStalled_StuckConsumer(t *testing.T) {
	p := fakeProbe(0.01, 0.0)
	if !p.isStalled(50, 0.0, 1.2, "prod") {
		t.Fatal("stuck consumer (ack=0, pub>0) MUST be stalled")
	}
}

func TestIsStalled_NoConsumerStillStalled(t *testing.T) {
	p := fakeProbe(0.01, 0.0)
	if !p.isStalled(10, 0.0, 0.5, "prod") {
		t.Fatal("no-consumer-but-publishing (ack=0, pub>0) MUST be stalled")
	}
}

func TestIsStalled_LowTrafficSteadyStateExempt(t *testing.T) {
	p := fakeProbe(0.01, 0.0)
	if p.isStalled(3, 0.0, 0.0, "prod") {
		t.Fatal("idle queue with no publishing must be exempt (publish_rate>0 clause)")
	}
}

func TestIsStalled_HealthyConsumerExempt(t *testing.T) {
	p := fakeProbe(0.01, 0.0)
	if p.isStalled(100, 5.0, 5.1, "prod") {
		t.Fatal("healthy consumer (ack=5/s) must not be stalled")
	}
}

func TestIsStalled_VHostBlacklistExempt(t *testing.T) {
	p := fakeProbe(0.01, 0.0, "bk_bknodeman")
	if p.isStalled(50, 0.0, 1.0, "bk_bknodeman") {
		t.Fatal("blacklisted vhost must be exempt from stalled rule")
	}
}

func TestIsStalled_ZeroMessagesNeverStalled(t *testing.T) {
	p := fakeProbe(0.01, 0.0)
	if p.isStalled(0, 0.0, 5.0, "prod") {
		t.Fatal("messages=0 must never be stalled regardless of rates")
	}
}

func TestIsStalled_NegativePubMinDisablesPubClause(t *testing.T) {
	p := fakeProbe(0.01, -1)
	// publish_rate = 0 but ack_rate is still below max → stalled fires
	if !p.isStalled(10, 0.0, 0.0, "prod") {
		t.Fatal("negative pub_min must disable the publish-rate clause")
	}
}

func TestMessageStatsRate_MissingSubobject(t *testing.T) {
	if r := messageStatsRate(map[string]interface{}{}, "ack_details"); r != 0 {
		t.Errorf("missing message_stats must yield 0, got %v", r)
	}
}

func TestMessageStatsRate_PresentRate(t *testing.T) {
	q := map[string]interface{}{
		"message_stats": map[string]interface{}{
			"ack_details": map[string]interface{}{"rate": 3.5},
		},
	}
	if r := messageStatsRate(q, "ack_details"); r != 3.5 {
		t.Errorf("ack_details.rate parse mismatch, got %v want 3.5", r)
	}
}

func TestMessageStatsRate_MissingSubkey(t *testing.T) {
	q := map[string]interface{}{
		"message_stats": map[string]interface{}{
			"ack_details": map[string]interface{}{"rate": 1.0},
		},
	}
	if r := messageStatsRate(q, "publish_details"); r != 0 {
		t.Errorf("missing publish_details must yield 0, got %v", r)
	}
}

func TestIsStalled_MessageStatsAbsentTreatedAsZeroRates(t *testing.T) {
	// Simulates a freshly-created queue: message_stats missing →
	// ack_rate=0, publish_rate=0 → publish_rate>0 clause exempts the queue
	// from the stalled rule even though messages > 0.
	p := fakeProbe(0.01, 0.0)
	q := map[string]interface{}{"messages": float64(5)}
	ack := messageStatsRate(q, "ack_details")
	pub := messageStatsRate(q, "publish_details")
	if p.isStalled(5, ack, pub, "prod") {
		t.Fatal("queue with absent message_stats must NOT be stalled (pub clause exempts)")
	}
}
