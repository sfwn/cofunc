package resource

import (
	"io"
	"net/http"
	"time"
)

type LogStdoutPrinter interface {
	PrintTitle()
	PrintSummary()
	Reset() error
}

// Resources contains some services that can be used by the driver and function.
// .e.g. logset service, cron service, httpserver service etc.
type Resources struct {
	Logwriter   io.Writer
	CronTrigger CronTrigger
	HttpTrigger HttpTrigger
}

// CronTrigger add and remove the cron job by trigger function, the CronTrigger is a resrouce for trigger.
type CronTrigger interface {
	Add(format string, ch chan<- time.Time) (interface{}, error)
	Remove(interface{}) error
}

// HttpTrigger add and remove the http handler by trigger function, the HttpTrigger is a resrouce for trigger.
type HttpTrigger interface {
	AddRoute(path string, handler func(w http.ResponseWriter, r *http.Request)) error
	RemoveRoute(path string) error
}
