package checker

import (
	"fmt"
	"strconv"

	"weops-inspect/config"
	"weops-inspect/model"
)

// CheckRabbitMQ produces CheckResults for RabbitMQ. All findings are Warn.
//
// Per design.md D4, ExceedingQueues / StalledQueues are already filtered by
// the collector (with vhost blacklist applied), so the checker just maps each
// element to a Warn CheckResult and backfills per-cell render statuses.
func CheckRabbitMQ(r *model.RabbitMQStatus, t config.Thresholds) []model.CheckResult {
	if r == nil {
		return nil
	}
	var results []model.CheckResult

	if r.Error != "" {
		results = append(results, model.CheckResult{
			Field: "rabbitmq.error", Value: r.Error, Status: model.StatusWarn,
		})
	}

	if r.QueuesError != "" {
		results = append(results, model.CheckResult{
			Field: "rabbitmq.queues_error", Value: r.QueuesError, Status: model.StatusWarn,
		})
	}

	if r.ClusterPartition {
		r.PartitionStatus = model.StatusWarn
		results = append(results, model.CheckResult{
			Field: "rabbitmq.cluster_partition", Value: "true", Status: model.StatusWarn,
		})
	} else {
		r.PartitionStatus = model.StatusOK
	}

	if r.AbnormalConnections > 0 {
		r.AbnormalConnStatus = model.StatusWarn
		results = append(results, model.CheckResult{
			Field:  "rabbitmq.abnormal_connections",
			Value:  fmt.Sprintf("%d", r.AbnormalConnections),
			Status: model.StatusWarn,
		})
	} else {
		r.AbnormalConnStatus = ""
	}

	for i := range r.NodeAlarms {
		a := &r.NodeAlarms[i]
		if a.MemAlarm {
			a.MemStatus = model.StatusWarn
			results = append(results, model.CheckResult{
				Field:  "rabbitmq.node." + a.Node + ".mem_alarm",
				Value:  "true",
				Status: model.StatusWarn,
			})
		} else {
			a.MemStatus = model.StatusOK
		}
		if a.DiskFreeAlarm {
			a.DiskFreeStatus = model.StatusWarn
			results = append(results, model.CheckResult{
				Field:  "rabbitmq.node." + a.Node + ".disk_free_alarm",
				Value:  "true",
				Status: model.StatusWarn,
			})
		} else {
			a.DiskFreeStatus = model.StatusOK
		}
	}

	backlogThr := fmt.Sprintf("> %d msgs", t.RabbitMQQueueBacklog)
	for i := range r.ExceedingQueues {
		q := &r.ExceedingQueues[i]
		q.MessageStatus = model.StatusWarn
		results = append(results, model.CheckResult{
			Field:     "rabbitmq." + q.VHost + "." + q.Queue + ".backlog",
			Value:     fmt.Sprintf("%d msgs / %d consumers", q.MessageCount, q.Consumers),
			Status:    model.StatusWarn,
			Threshold: backlogThr,
		})
	}

	stalledThr := stalledThresholdLabel(t)
	for i := range r.StalledQueues {
		q := &r.StalledQueues[i]
		q.ConsumerStatus = model.StatusWarn
		q.MessageStatus = model.StatusWarn
		results = append(results, model.CheckResult{
			Field: "rabbitmq." + q.VHost + "." + q.Queue + ".stalled",
			Value: fmt.Sprintf("%d msgs / %d consumers / ack=%s/s / pub=%s/s",
				q.MessageCount, q.Consumers, formatRate(q.AckRate), formatRate(q.PublishRate)),
			Status:    model.StatusWarn,
			Threshold: stalledThr,
		})
	}

	return results
}

// stalledThresholdLabel formats the human-readable threshold tail shown in the
// alert email. The publish-rate clause is dropped when disabled (min<0) so
// operators see only the ack-rate condition that actually fires.
func stalledThresholdLabel(t config.Thresholds) string {
	if t.RabbitMQStalledPublishRateMin < 0 {
		return fmt.Sprintf("ack < %s/s", formatRate(t.RabbitMQStalledAckRateMax))
	}
	return fmt.Sprintf("ack < %s/s, pub > %s/s",
		formatRate(t.RabbitMQStalledAckRateMax),
		formatRate(t.RabbitMQStalledPublishRateMin))
}

// formatRate renders a per-second rate compactly: drops trailing zeros so the
// reader sees "0", "0.01", "1.2" instead of "0.000000", "0.010000", "1.200000".
func formatRate(r float64) string {
	return strconv.FormatFloat(r, 'f', -1, 64)
}
