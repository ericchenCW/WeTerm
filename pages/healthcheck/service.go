package healthcheck

import (
	"fmt"
	"sort"
	"weterm/pages/template/table"

	capi "github.com/hashicorp/consul/api"
	"github.com/rs/zerolog/log"
)

type ServiceHealth struct {
	BaseHealthChecker
	c *capi.Client
}

func NewServiceHealth() ServiceHealth {
	config := capi.DefaultConfig()
	client, err := capi.NewClient(config)
	if err != nil {
		panic(err)
	}
	return ServiceHealth{
		c: client,
	}
}

func (h ServiceHealth) Detail(serviceID string) string {
	filterExpression := fmt.Sprintf(`ServiceID == "%s"`, serviceID)
	q := &capi.QueryOptions{
		Filter: filterExpression,
	}
	check, _, err := h.c.Health().State("any", q)
	if err != nil {
		log.Logger.Err(err).Msg("Get Consul Services Detail Error")
	}
	if len(check) < 1 {
		return "None Value"
	}
	return check[0].Output
}

func (h ServiceHealth) Check() table.TableData {
	result := table.TableData{Header: table.Header{
		table.HeaderColumn{Name: "Name"},
		table.HeaderColumn{Name: "ID"},
		table.HeaderColumn{Name: "Node"},
		table.HeaderColumn{Name: "Type"},
		table.HeaderColumn{Name: "Status"},
	}}
	services, _, err := h.c.Health().State("any", nil)
	if err != nil {
		log.Logger.Err(err).Msg("Get Consul Services Error")
	}
	for i := range services {
		s := services[i]
		row := table.NewRow(5)
		if s.ServiceName == "" {
			row.Fields[0] = "[yellow]consul"
		} else {
			row.Fields[0] = "[yellow]" + s.ServiceName
		}
		if s.ServiceID == "" {
			row.Fields[1] = "[aqua]consul"
		} else {
			row.Fields[1] = s.ServiceID
		}
		row.Fields[2] = "[yellow]" + s.Node
		row.Fields[3] = "[aqua]" + s.Type
		row.Fields[4] = h.buildColorStatus(s.Status)
		// row.Fields[4] = "[white]" + s.Output
		result.Rows = append(result.Rows, row)
	}
	sort.Slice(result.Rows, func(i, j int) bool {
		if result.Rows[i].Fields[4] == result.Rows[j].Fields[4] {
			return result.Rows[i].Fields[0] < result.Rows[j].Fields[0]
		}
		return result.Rows[i].Fields[4] >= result.Rows[j].Fields[4]
	})
	return result
}

func (h ServiceHealth) buildColorStatus(s string) string {
	switch s {
	case "passing":
		return "[green]" + s
	case "warning":
		return "[orange]" + s
	case "critical":
		return "[red]" + s
	default:
		return "[white]None"
	}
}
