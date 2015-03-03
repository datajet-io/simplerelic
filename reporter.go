package simplerelic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	// url of the NewRelic plugin API
	newrelicURL = "https://platform-api.newrelic.com/platform/v1/metrics"

	// default GUID that associate the metrics with a NewRelic plugin
	defaultGUID = "com.github.domenp.SimpleRelic"

	// how often we send the metrics to NewRelic
	reportingFreq = time.Duration(60) * time.Second

	// for debugging purposes sending metrics can be disabled
	sendMetrics = true
)

var (
	// Log is a logger used in the package
	Log *log.Logger

	// NewRelic GUID for creating the NewRelic plugin
	Guid string

	httpClient = &http.Client{Timeout: 10 * time.Second}
)

func init() {
	Guid = defaultGUID
	Log = log.New(os.Stderr, "[simplerelic] ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Reporter keeps track of the app metrics and sends them to NewRelic
type Reporter struct {
	Metrics  []AppMetric
	host     string
	pid      int
	guid     string
	duration int
	version  string
	appName  string
	licence  string
	verbose  bool
}

type newRelicData struct {
	Agent      *newRelicAgent       `json:"agent"`
	Components []*newRelicComponent `json:"components"`
}

type newRelicAgent struct {
	Host    string `json:"host"`
	Pid     int    `json:"pid"`
	Version string `json:"version"`
}

type newRelicComponent struct {
	Name     string             `json:"name"`
	Guid     string             `json:"guid"`
	Duration int                `json:"duration"`
	Metrics  map[string]float32 `json:"metrics"`
}

// NewReporter creates a new Reporter
func NewReporter(appName string, licence string, verbose bool) (*Reporter, error) {

	host, err := os.Hostname()
	if err != nil {
		return nil, errors.New("Can not get hostname")
	}

	pid := os.Getpid()

	if licence == "" {
		return nil, errors.New("Please specify Newrelic licence")
	}

	reporter := &Reporter{
		host:     host,
		pid:      pid,
		guid:     Guid,
		duration: 60,
		appName:  appName,
		licence:  licence,
		version:  "1.0.0",
		verbose:  verbose,
		Metrics:  make([]AppMetric, 0, 5),
	}

	return reporter, nil
}

// Start sending metrics to NewRelic
func (reporter *Reporter) Start() {

	ticker := time.NewTicker(reportingFreq)
	quit := make(chan struct{})
	go func() {

		defer func() {
			if r := recover(); r != nil {
				Log.Println("SimpleRelic reporter crashed")
			}
		}()

		for {
			select {
			case <-ticker.C:
				reporter.sendMetrics()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

// AddMetric adds a new metric to be reported
func (reporter *Reporter) AddMetric(metric AppMetric) {
	reporter.Metrics = append(reporter.Metrics, metric)
}

// extract and send metrics to NewRelic
func (reporter *Reporter) sendMetrics() {

	reqData := reporter.prepareReqData()

	// extract all metrics to be sent to NewRelic
	// from the AppMetric data structure
	for _, metrics := range reporter.Metrics {
		for name, value := range metrics.ValueMap() {
			reqData.Components[0].Metrics[name] = value
		}
	}

	b, err := json.Marshal(reqData)
	if err != nil {
		fmt.Errorf("error marshaling json")
	}

	if reporter.verbose {
		var out bytes.Buffer
		json.Indent(&out, b, "", "\t")
		Log.Println("sending metrics to NewRelic")
		Log.Println(out.String())
	}

	if sendMetrics {
		reporter.doRequest(b)
	}
}

func (reporter *Reporter) prepareReqData() *newRelicData {
	reqData := &newRelicData{
		Agent: &newRelicAgent{
			Host:    reporter.host,
			Pid:     reporter.pid,
			Version: reporter.version,
		},
		Components: []*newRelicComponent{
			&newRelicComponent{
				Name:     reporter.appName,
				Guid:     reporter.guid,
				Duration: reporter.duration,
				Metrics:  make(map[string]float32),
			},
		},
	}

	reqData.Components[0] = &newRelicComponent{
		Name:     reporter.appName,
		Guid:     reporter.guid,
		Duration: reporter.duration,
		Metrics:  make(map[string]float32),
	}

	return reqData
}

func (reporter *Reporter) doRequest(json []byte) {
	req, err := http.NewRequest("POST", newrelicURL, bytes.NewReader(json))
	if err != nil {
		Log.Println("error setting up newrelic request")
	}
	req.Header.Set("X-License-Key", reporter.licence)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		Log.Println("Post request to NewRelic failed")
		Log.Println(err)
		return
	}
	defer resp.Body.Close()

	if reporter.verbose {
		responseJSON, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			Log.Println("reading of NewRelic response failed")
		}
		Log.Println("response from NewRelic")
		Log.Println(string(responseJSON))
	}

	if resp.StatusCode != http.StatusOK {
		Log.Printf("Error in request to NewRelic, status code %d", resp.StatusCode)
	}
}
