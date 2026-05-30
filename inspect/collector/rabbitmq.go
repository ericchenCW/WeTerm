package collector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"weops-inspect/config"
	"weops-inspect/model"
)

// CollectRabbitMQ collects RabbitMQ cluster status via the Management HTTP API.
// Implementation still calls the local `curl` binary by design; only the
// network layer is wrapped in the Probe framework for ctx/error-class parity.
func CollectRabbitMQ(ctx context.Context, cfg *config.Config) *model.RabbitMQStatus {
	if len(cfg.RabbitMQIPs) == 0 {
		return nil
	}
	if _, err := exec.LookPath("curl"); err != nil {
		return &model.RabbitMQStatus{Error: "curl CLI not available", ErrorClass: string(ErrUnknown)}
	}

	host := cfg.RabbitMQIPs[0]
	port := "15672"
	user := cfg.Creds.RabbitMQUser
	pass := cfg.Creds.RabbitMQPassword

	status := &model.RabbitMQStatus{}
	target := net.JoinHostPort(host, port)

	apiTimeout := cfg.RabbitMQAPITimeoutSec
	if apiTimeout <= 0 {
		apiTimeout = 60
	}
	stalledVHostBL := make(map[string]bool, len(cfg.Thresholds.RabbitMQStalledVHostBlacklist))
	for _, v := range cfg.Thresholds.RabbitMQStalledVHostBlacklist {
		stalledVHostBL[v] = true
	}
	probe := &rmqProbe{
		host: host, port: port, user: user, pass: pass,
		target: target, status: status,
		backlogThreshold:    cfg.Thresholds.RabbitMQQueueBacklog,
		stalledAckRateMax:   cfg.Thresholds.RabbitMQStalledAckRateMax,
		stalledPubRateMin:   cfg.Thresholds.RabbitMQStalledPublishRateMin,
		stalledVHostBL:      stalledVHostBL,
		apiTimeoutSec:       apiTimeout,
	}
	// 4 个 endpoint 串行调用,各自最多 apiTimeout 秒;给框架一个略宽松的总 deadline,
	// 避免 RunProbe 默认 5s 兜底 kill 掉 curl ("signal: killed")。
	probeCtx, cancel := context.WithTimeout(ctx, time.Duration(apiTimeout*4+5)*time.Second)
	defer cancel()
	RunProbe(probeCtx, probe)
	return status
}

type rmqProbe struct {
	host, port, user, pass string
	target                 string
	status                 *model.RabbitMQStatus
	backlogThreshold       int
	// Stalled-queue thresholds: ack_rate < max AND publish_rate > min triggers.
	// A negative pubRateMin disables the publish-rate clause entirely (ack
	// alone gates the rule).
	stalledAckRateMax float64
	stalledPubRateMin float64
	// vhosts whose stalled queues should not raise an alert. Backlog alerts
	// still apply even for blacklisted vhosts.
	stalledVHostBL map[string]bool
	apiTimeoutSec  int
}

func (p *rmqProbe) Name() string { return "rabbitmq" }

func (p *rmqProbe) Run(ctx context.Context) ProbeResult {
	if nodesJSON, err := rmqAPI(ctx, p.host, p.port, p.user, p.pass, "nodes?columns=name,mem_alarm,disk_free_alarm,partitions,uptime", p.apiTimeoutSec); err != nil {
		p.status.Error = err.Error()
		p.status.ErrorClass = string(curlErrClass(err))
		return ProbeResult{Target: p.target, Err: err, ErrClass: curlErrClass(err)}
	} else if nodesJSON != nil {
		var nodes []map[string]interface{}
		if json.Unmarshal(nodesJSON, &nodes) == nil {
			for _, n := range nodes {
				alarm := model.RabbitMQAlarm{Node: jsonStr(n["name"])}
				if v, ok := n["mem_alarm"].(bool); ok {
					alarm.MemAlarm = v
				}
				if v, ok := n["disk_free_alarm"].(bool); ok {
					alarm.DiskFreeAlarm = v
				}
				if alarm.MemAlarm || alarm.DiskFreeAlarm {
					p.status.NodeAlarms = append(p.status.NodeAlarms, alarm)
				}
				if parts, ok := n["partitions"].([]interface{}); ok && len(parts) > 0 {
					p.status.ClusterPartition = true
				}
				if p.status.Uptime == "" {
					if uptimeMs, ok := n["uptime"].(float64); ok {
						secs := int(uptimeMs / 1000)
						days := secs / 86400
						hours := (secs % 86400) / 3600
						mins := (secs % 3600) / 60
						p.status.Uptime = fmt.Sprintf("%dd %dh %dm", days, hours, mins)
					}
				}
			}
		}
	}

	if connsJSON, err := rmqAPI(ctx, p.host, p.port, p.user, p.pass, "connections?columns=state", p.apiTimeoutSec); err == nil && connsJSON != nil {
		var conns []map[string]interface{}
		if json.Unmarshal(connsJSON, &conns) == nil {
			p.status.TotalConnections = len(conns)
			for _, c := range conns {
				state := jsonStr(c["state"])
				if state != "running" && state != "" {
					p.status.AbnormalConnections++
				}
			}
		}
	}

	if chansJSON, err := rmqAPI(ctx, p.host, p.port, p.user, p.pass, "channels?columns=name", p.apiTimeoutSec); err == nil && chansJSON != nil {
		var chans []interface{}
		if json.Unmarshal(chansJSON, &chans) == nil {
			p.status.TotalChannels = len(chans)
		}
	}

	// Keep disable_stats=true (avoids returning the full per-queue stats blob)
	// but explicitly request message_stats.ack_details.rate and
	// publish_details.rate via columns so the rolling consumer-throughput
	// signal is available for the stalled-queue rule. RabbitMQ honors the
	// columns whitelist even with disable_stats; missing message_stats on
	// queues that have never had deliver/ack activity is expected and parsed
	// as 0.0 (handled by jsonFloat / messageStatsRate).
	queuesJSON, qErr := rmqAPI(ctx, p.host, p.port, p.user, p.pass, "queues?disable_stats=true&enable_queue_totals=true&columns=name,vhost,messages,consumers,durable,message_stats.ack_details.rate,message_stats.publish_details.rate", p.apiTimeoutSec)
	if qErr != nil {
		p.status.QueuesError = qErr.Error()
	} else if queuesJSON != nil {
		var queues []map[string]interface{}
		if json.Unmarshal(queuesJSON, &queues) == nil {
			summaryByVHost := map[string]*model.RabbitMQVHostSummary{}
			for _, q := range queues {
				vhost := jsonStr(q["vhost"])
				name := jsonStr(q["name"])
				if vhost == "bk_usermgr" || strings.HasPrefix(name, "celeryev") {
					continue
				}
				msgs := jsonInt(q["messages"])
				consumers := jsonInt(q["consumers"])
				ackRate := messageStatsRate(q, "ack_details")
				pubRate := messageStatsRate(q, "publish_details")

				agg, ok := summaryByVHost[vhost]
				if !ok {
					agg = &model.RabbitMQVHostSummary{VHost: vhost}
					summaryByVHost[vhost] = agg
				}
				agg.Queues++
				agg.Messages += msgs
				agg.Consumers += consumers

				queueInfo := model.RabbitMQQueue{
					VHost:        vhost,
					Queue:        name,
					MessageCount: msgs,
					Consumers:    consumers,
					AckRate:      ackRate,
					PublishRate:  pubRate,
				}
				if v, ok := q["durable"].(bool); ok {
					queueInfo.Durable = v
				}
				if msgs >= p.backlogThreshold {
					p.status.ExceedingQueues = append(p.status.ExceedingQueues, queueInfo)
				}
				if p.isStalled(msgs, ackRate, pubRate, vhost) {
					p.status.StalledQueues = append(p.status.StalledQueues, queueInfo)
				}
			}
			vhosts := make([]string, 0, len(summaryByVHost))
			for v := range summaryByVHost {
				vhosts = append(vhosts, v)
			}
			sort.Strings(vhosts)
			for _, v := range vhosts {
				p.status.VHostSummary = append(p.status.VHostSummary, *summaryByVHost[v])
			}
		}
	}

	return ProbeResult{Target: p.target}
}

func rmqAPI(ctx context.Context, host, port, user, pass, endpoint string, timeoutSec int) ([]byte, error) {
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	url := fmt.Sprintf("http://%s:%s/api/%s", host, port, endpoint)
	authHeader := fmt.Sprintf("%s:%s", user, pass)

	args := []string{"-s", "-S", "--max-time", strconv.Itoa(timeoutSec), "-u", authHeader, "-H", "Accept: application/json", url}
	out, err := exec.CommandContext(ctx, "curl", args...).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("%s: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, err
	}
	trimmed := strings.TrimSpace(string(out))
	if len(trimmed) == 0 || (trimmed[0] != '[' && trimmed[0] != '{') {
		return nil, fmt.Errorf("rabbitmq api %s: non-JSON response", endpoint)
	}
	return out, nil
}

// messageStatsRate pulls `message_stats.<subkey>.rate` out of a queue's JSON
// object. Absent message_stats / absent subkey / non-numeric rate all surface
// as 0.0 — for new queues that have never seen deliver/ack activity the API
// omits the whole sub-object, and treating that as 0 is correct: combined
// with the publish_rate>min clause, such queues are naturally exempt from the
// stalled rule.
func messageStatsRate(q map[string]interface{}, subkey string) float64 {
	ms, ok := q["message_stats"].(map[string]interface{})
	if !ok {
		return 0
	}
	sub, ok := ms[subkey].(map[string]interface{})
	if !ok {
		return 0
	}
	return jsonFloat(sub["rate"])
}

// isStalled returns true when a queue meets the stalled-queue rule:
// messages>0 AND ack_rate<max AND publish_rate>min AND vhost not blacklisted.
// A negative configured pubRateMin disables the publish-rate clause (only
// ack_rate gates the rule then).
func (p *rmqProbe) isStalled(msgs int, ackRate, pubRate float64, vhost string) bool {
	if msgs <= 0 {
		return false
	}
	if ackRate >= p.stalledAckRateMax {
		return false
	}
	if p.stalledPubRateMin >= 0 && pubRate <= p.stalledPubRateMin {
		return false
	}
	if p.stalledVHostBL[vhost] {
		return false
	}
	return true
}

// curlErrClass 从 curl 的 exit code 反推 ErrorClass。
func curlErrClass(err error) ErrorClass {
	if err == nil {
		return ErrNone
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		return Classify(err)
	}
	switch ee.ExitCode() {
	case 6, 7: // couldn't resolve / couldn't connect
		return ErrNetwork
	case 22: // HTTP non-2xx
		return ErrProtocol
	case 28: // operation timed out
		return ErrTimeout
	case 67: // login failed
		return ErrAuth
	}
	return ErrUnknown
}
