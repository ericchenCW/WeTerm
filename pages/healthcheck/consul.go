package healthcheck

import (
	"sort"
	"strconv"
	"weterm/pages/template/table"

	"github.com/rs/zerolog/log"

	capi "github.com/hashicorp/consul/api"
)

type ConsulHealth struct {
	BaseHealthChecker
	c *capi.Client
}

func NewConsulHealth() ConsulHealth {
	config := capi.DefaultConfig()
	client, err := capi.NewClient(config)
	if err != nil {
		panic(err)
	}
	return ConsulHealth{
		c: client,
	}
}

func (h ConsulHealth) Check() table.TableData {
	result := table.TableData{Header: table.Header{
		table.HeaderColumn{Name: "Addr"},
		table.HeaderColumn{Name: "Name"},
		table.HeaderColumn{Name: "Port"},
		table.HeaderColumn{Name: "DelegateCur"},
		table.HeaderColumn{Name: "DelegateMax"},
		table.HeaderColumn{Name: "DelegateMin"},
		table.HeaderColumn{Name: "ProtocolCur"},
		table.HeaderColumn{Name: "ProtocolMax"},
		table.HeaderColumn{Name: "ProtocolMin"},
		table.HeaderColumn{Name: "Status"},
	}}
	// 获取consul集群状态
	members, err := h.c.Agent().Members(false)
	//    Agent状态描述
	// 	  AgentMemberNone    = 0
	//	  AgentMemberAlive   = 1
	//	  AgentMemberLeaving = 2
	//	  AgentMemberLeft    = 3
	//	  AgentMemberFailed  = 4
	//    这里简单处理，不正常的都返回Error
	if err != nil {
		log.Logger.Err(err).Msg("Get Consul Members Error")
	}
	for i := range members {
		m := members[i]
		row := table.NewRow(10)
		row.Fields[0] = m.Addr
		row.Fields[1] = m.Name
		row.Fields[2] = strconv.FormatUint(uint64(m.Port), 10)
		row.Fields[3] = strconv.FormatUint(uint64(m.DelegateCur), 10)
		row.Fields[4] = strconv.FormatUint(uint64(m.DelegateMax), 10)
		row.Fields[5] = strconv.FormatUint(uint64(m.DelegateMin), 10)
		row.Fields[6] = strconv.FormatUint(uint64(m.ProtocolCur), 10)
		row.Fields[7] = strconv.FormatUint(uint64(m.ProtocolMax), 10)
		row.Fields[8] = strconv.FormatUint(uint64(m.ProtocolMin), 10)
		row.Fields[9] = h.buildColorStatus(m.Status)
		result.Rows = append(result.Rows, row)
	}
	sort.Slice(result.Rows, func(i, j int) bool {
		return result.Rows[i].Fields[0] < result.Rows[j].Fields[0]
	})
	return result
}

func (h ConsulHealth) buildColorStatus(status int) string {
	switch status {
	case 0:
		return "[white]None"
	case 1:
		return "[green]Alive"
	case 2:
		return "[orange]Leaving"
	case 3:
		return "[red]Left"
	case 4:
		return "[red]Failed"
	default:
		return "[white]None"
	}
}
