package healthcheck

import (
	capi "github.com/hashicorp/consul/api"
)

type DatalinkHealth struct {
	BaseHealthChecker
	consul *capi.Client
	// kafkaConsumer *kafka.Consumer
	metaTopic     string
	metaPartition int
}

func NewDatainlkHealth() DatalinkHealth {
	config := capi.DefaultConfig()
	client, err := capi.NewClient(config)
	if err != nil {
		panic(err)
	}
	// kafkaConfig := kafka.ConfigMap{
	// 	"bootstrap.servers":  os.Getenv("kafka.service.consul:9092"),
	// 	"compression.codec":  "none",
	// 	"batch.num.messages": "1",
	// 	"group.id":           "bkmonitorv3_transfer0bkmonitor_10010",
	// 	"auto.offset.reset":  "latest",
	// }
	// consumer, _ := kafka.NewConsumer(&kafkaConfig)
	return DatalinkHealth{
		consul: client,
		// kafkaConsumer: consumer,
		metaTopic:     "0bkmonitor_10010",
		metaPartition: 0,
	}
}

// TODO 实现检查逻辑
func (d DatalinkHealth) Check() []HealthResult {
	result := []HealthResult{}
	monitorServices := []string{
		"gse-data",
		"kafka",
		"influxdb",
		"bkmonitorv3",
	}
	for _, service := range monitorServices {
		print(service)
	}
	return result
}
