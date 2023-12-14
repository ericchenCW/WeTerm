package healthcheck

import "fmt"

const (
	Healthy = "[green] ✔ Healthy [aqua] %s"
	Warning = "[yellow] ⚠ Warning %s"
	Error   = "[red] ✖ Error!! [white] %s"
)

type HealthResult struct {
	status  string
	message string
}

type Health interface {
	Check() []HealthResult
	Print(HealthResult) string
}

type BaseHealthChecker struct{}

func (b BaseHealthChecker) Print(r HealthResult) string {
	return fmt.Sprintf(r.status, r.message)
}
