package checker

import (
	"strings"
	"testing"

	"weops-inspect/config"
	"weops-inspect/model"
)

var rmqTestThresholds = config.Thresholds{
	RabbitMQQueueBacklog:          10000,
	RabbitMQStalledAckRateMax:     0.01,
	RabbitMQStalledPublishRateMin: 0.0,
}

func TestCheckRabbitMQ_Nil(t *testing.T) {
	if got := CheckRabbitMQ(nil, rmqTestThresholds); got != nil {
		t.Errorf("want nil, got %v", got)
	}
}

func TestCheckRabbitMQ_AllProblems(t *testing.T) {
	r := &model.RabbitMQStatus{
		Error:               "boom",
		QueuesError:         "queues 500",
		ClusterPartition:    true,
		AbnormalConnections: 2,
		NodeAlarms:          []model.RabbitMQAlarm{{Node: "rabbit@n1", MemAlarm: true, DiskFreeAlarm: true}},
		ExceedingQueues:     []model.RabbitMQQueue{{VHost: "v1", Queue: "celery", MessageCount: 360547}},
		StalledQueues: []model.RabbitMQQueue{{
			VHost: "v1", Queue: "default", MessageCount: 8, Consumers: 2,
			AckRate: 0.0, PublishRate: 1.2,
		}},
	}
	got := CheckRabbitMQ(r, rmqTestThresholds)
	if len(got) < 7 {
		t.Errorf("want >=7 warns, got %d: %v", len(got), got)
	}
	for _, c := range got {
		if c.Status != model.StatusWarn {
			t.Errorf("want warn, got %v", c)
		}
	}
	if r.PartitionStatus != model.StatusWarn {
		t.Errorf("PartitionStatus = %v", r.PartitionStatus)
	}
	if r.ExceedingQueues[0].MessageStatus != model.StatusWarn {
		t.Errorf("MessageStatus = %v", r.ExceedingQueues[0].MessageStatus)
	}
	if r.StalledQueues[0].ConsumerStatus != model.StatusWarn {
		t.Errorf("ConsumerStatus = %v", r.StalledQueues[0].ConsumerStatus)
	}
}

func TestCheckRabbitMQ_AllHealthy(t *testing.T) {
	r := &model.RabbitMQStatus{}
	got := CheckRabbitMQ(r, rmqTestThresholds)
	if len(got) != 0 {
		t.Errorf("want no warns, got %v", got)
	}
}

func TestCheckRabbitMQ_BacklogFieldFormat(t *testing.T) {
	r := &model.RabbitMQStatus{
		ExceedingQueues: []model.RabbitMQQueue{{VHost: "prod_bk_monitorv3", Queue: "celery", MessageCount: 360547}},
	}
	got := CheckRabbitMQ(r, rmqTestThresholds)
	if len(got) != 1 {
		t.Fatalf("got %v", got)
	}
	if got[0].Field != "rabbitmq.prod_bk_monitorv3.celery.backlog" {
		t.Errorf("field = %q", got[0].Field)
	}
}

func TestCheckRabbitMQ_StalledFieldAndValueFormat(t *testing.T) {
	r := &model.RabbitMQStatus{
		StalledQueues: []model.RabbitMQQueue{{
			VHost: "prod", Queue: "q1", MessageCount: 50, Consumers: 2,
			AckRate: 0.0, PublishRate: 1.2,
		}},
	}
	got := CheckRabbitMQ(r, rmqTestThresholds)
	if len(got) != 1 {
		t.Fatalf("got %v", got)
	}
	c := got[0]
	if c.Field != "rabbitmq.prod.q1.stalled" {
		t.Errorf("Field = %q", c.Field)
	}
	for _, want := range []string{"50 msgs", "2 consumers", "ack=0/s", "pub=1.2/s"} {
		if !strings.Contains(c.Value, want) {
			t.Errorf("Value = %q, missing %q", c.Value, want)
		}
	}
	if c.Threshold != "ack < 0.01/s, pub > 0/s" {
		t.Errorf("Threshold = %q", c.Threshold)
	}
}

func TestCheckRabbitMQ_StalledThresholdOmitsPubWhenDisabled(t *testing.T) {
	r := &model.RabbitMQStatus{
		StalledQueues: []model.RabbitMQQueue{{
			VHost: "v", Queue: "q", MessageCount: 1,
		}},
	}
	thr := config.Thresholds{
		RabbitMQQueueBacklog:          10000,
		RabbitMQStalledAckRateMax:     0.05,
		RabbitMQStalledPublishRateMin: -1,
	}
	got := CheckRabbitMQ(r, thr)
	if len(got) != 1 {
		t.Fatalf("got %v", got)
	}
	if got[0].Threshold != "ack < 0.05/s" {
		t.Errorf("Threshold = %q, want %q", got[0].Threshold, "ack < 0.05/s")
	}
}

func TestCheckRabbitMQ_BacklogAndStalledCoexist(t *testing.T) {
	r := &model.RabbitMQStatus{
		ExceedingQueues: []model.RabbitMQQueue{{VHost: "v", Queue: "q", MessageCount: 50000}},
		StalledQueues:   []model.RabbitMQQueue{{VHost: "v", Queue: "q", MessageCount: 50000, Consumers: 1}},
	}
	got := CheckRabbitMQ(r, rmqTestThresholds)
	var backlog, stalled int
	for _, c := range got {
		if strings.HasSuffix(c.Field, ".backlog") {
			backlog++
		}
		if strings.HasSuffix(c.Field, ".stalled") {
			stalled++
		}
	}
	if backlog != 1 || stalled != 1 {
		t.Errorf("want 1 backlog + 1 stalled, got backlog=%d stalled=%d (%v)", backlog, stalled, got)
	}
}
