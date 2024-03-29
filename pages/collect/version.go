package collect

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"weterm/pages/template/table"
	"weterm/utils"

	"github.com/rs/zerolog/log"
)

var paasBackendVersionSQL = `SELECT code,name,version FROM bksuite_common.production_info`
var saasVersibSQL = `select code,name,version from open_paas.paas_saas_app join open_paas.paas_saas_app_version psav on paas_saas_app.current_version_id=psav.id`

//go:embed asserts/version_images.py
var imageScript string

type VersionRow struct {
	ID      string `json:"Image"`
	Version string `json:"Tag"`
	Name    string `json:"-"`
}

type ImagesVersionRow struct {
	VersionRow
	Node string
	Name string `json:"Name"`
}

var paasVersionHeader = table.Header{
	table.HeaderColumn{Name: "Name"},
	table.HeaderColumn{Name: "ID"},
	table.HeaderColumn{Name: "Version"},
}

var imageVersionHeader = table.Header{
	table.HeaderColumn{Name: "Name"},
	table.HeaderColumn{Name: "ID"},
	table.HeaderColumn{Name: "Version"},
	table.HeaderColumn{Name: "Node"},
}

func ParseRow(rows *sql.Rows) VersionRow {
	var version VersionRow
	err := rows.Scan(&version.ID, &version.Name, &version.Version)
	if err != nil {
		log.Logger.Err(err).Msg("Parse row failed")
	}
	return version
}

func toRow(v VersionRow) table.Row {
	row := table.NewRow(3)
	row.Fields[0] = v.Name
	row.Fields[1] = v.ID
	row.Fields[2] = v.Version
	return row
}

func toTable(rows []VersionRow) table.TableData {
	result := table.TableData{Header: paasVersionHeader}
	for v := range rows {
		result.Rows = append(result.Rows, toRow(rows[v]))
	}
	return result
}

func toImageTable(rows []ImagesVersionRow) table.TableData {
	result := table.TableData{Header: imageVersionHeader}
	for v := range rows {
		r := table.NewRow(4)
		r.Fields[0] = "[aqua]" + rows[v].Name
		r.Fields[1] = "[yellow]" + rows[v].ID
		r.Fields[2] = "[aqua]" + rows[v].Version
		r.Fields[3] = "[yellow]" + rows[v].Node
		result.Rows = append(result.Rows, r)
	}
	return result
}

func GetAppVersion() table.TableData {
	var result []VersionRow
	backendVersion, err := utils.MysqlQuery[VersionRow](paasBackendVersionSQL, ParseRow)
	if err != nil {
		log.Logger.Err(err).Msg("Query failed")
	}
	result = append(result, backendVersion...)
	log.Logger.Debug().Any("version", backendVersion).Msg("Query success")
	saasVersion, err := utils.MysqlQuery[VersionRow](saasVersibSQL, ParseRow)
	if err != nil {
		log.Logger.Err(err).Msg("Query failed")
	}
	result = append(result, saasVersion...)
	log.Logger.Debug().Any("version", saasVersion).Msg("Query success")
	return toTable(result)
}

func GetImageVersion() table.TableData {
	var result []ImagesVersionRow
	hosts := strings.Split(os.Getenv("ALL_IP_COMMA"), ",")
	ch := make(chan []ImagesVersionRow, len(hosts))
	var wg sync.WaitGroup
	for host := range hosts {
		wg.Add(1)
		go func(s string) {
			raw := utils.RunSSH(s, imageScript, "python")
			var arr []ImagesVersionRow
			err := json.Unmarshal(raw, &arr)
			if err != nil {
				log.Logger.Err(err).Msg("Unmarshal Docker images failed")
			}
			defer wg.Done()
			for a := range arr {
				arr[a].Node = s
			}
			ch <- arr
		}(hosts[host])
	}
	wg.Wait()
	close(ch)
	for res := range ch {
		result = append(result, res...)
	}
	return toImageTable(result)
}
