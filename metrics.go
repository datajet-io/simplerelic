package simplerelic

import (
	"errors"
	"sync"
	"time"
)

// AppMetric is an interface for metrics reported to NewRelic
type AppMetric interface {

	// Update all the values that will be reported (or be used in calculation).
	// For http metrics called on every request (for example in gin middleware)
	Update(params map[string]interface{}) error

	// ValueMap extracts all values from AppMetric data structures
	// to be reported to NewRelic.
	// A single AppMetric can produce multiple metrics as perceived by NewRelic
	// Consult NewRelic plugin API naming guidelines
	// before creating a new metric.
	//
	// Note that this function is also responsible for clearing the values
	// after they have been reported.
	ValueMap() map[string]float32
}

const (
	unknownEndpoint = "other"
)

// StandardMetric is a base for metrics dealing with endpoints
type StandardMetric struct {
	endpoints       map[string]func(urlPath string) bool
	reqCount        map[string]int
	lock            sync.RWMutex
	namePrefix      string
	allEPNamePrefix string
	metricUnit      string
}

func (m *StandardMetric) initReqCount() {
	// initialize the metrics
	for endpoint := range m.endpoints {
		m.reqCount[endpoint] = 0
	}
	m.reqCount[unknownEndpoint] = 0
}

func (m *StandardMetric) endpointName(params map[string]interface{}) string {
	endpointName, ok := params["endpointName"]
	if !ok {
		return unknownEndpoint
	}

	return endpointName.(string)
}

/************************************
 * requests per endpoint
 ***********************************/

// ReqPerEndpoint holds number of requests per endpoint
type ReqPerEndpoint struct {
	*StandardMetric
}

// NewReqPerEndpoint creates new ReqPerEndpoint metric
func NewReqPerEndpoint() *ReqPerEndpoint {

	metric := &ReqPerEndpoint{
		StandardMetric: &StandardMetric{
			reqCount:        make(map[string]int),
			namePrefix:      "Component/ReqPerEndpoint/",
			allEPNamePrefix: "Component/Req/overall",
			metricUnit:      "[requests]",
		},
	}

	metric.initReqCount()

	return metric
}

// Update the metric values
func (m *ReqPerEndpoint) Update(params map[string]interface{}) error {
	endpointName := m.endpointName(params)
	m.lock.Lock()
	m.reqCount[endpointName]++
	m.lock.Unlock()

	return nil
}

// ValueMap extract all the metrics to be reported
func (m *ReqPerEndpoint) ValueMap() map[string]float32 {

	metricMap := make(map[string]float32)

	m.lock.Lock()
	defer m.lock.Unlock()

	var numReqAllEndpoints int
	for endpoint, value := range m.reqCount {
		metricName := m.namePrefix + endpoint + m.metricUnit
		metricMap[metricName] = float32(value)

		numReqAllEndpoints += value
	}

	m.reqCount = make(map[string]int)

	metricMap[m.allEPNamePrefix+m.metricUnit] = float32(numReqAllEndpoints)

	return metricMap
}

/**************************************************
* Error rate per endpoint
**************************************************/

// ErrorRatePerEndpoint holds the percentage of error requests per endpoint
type ErrorRatePerEndpoint struct {
	*StandardMetric
	errorCount map[string]int
}

// NewErrorRatePerEndpoint creates new POEPerEndpoint metric
func NewErrorRatePerEndpoint() *ErrorRatePerEndpoint {

	metric := &ErrorRatePerEndpoint{
		StandardMetric: &StandardMetric{
			reqCount:        make(map[string]int),
			namePrefix:      "Component/ErrorRatePerEndpoint/",
			allEPNamePrefix: "Component/ErrorRate/overall",
			metricUnit:      "[percent]",
		},
		errorCount: make(map[string]int),
	}

	// initialize the metrics
	metric.initReqCount()
	for endpoint := range metric.endpoints {
		metric.errorCount[endpoint] = 0
	}
	metric.errorCount[unknownEndpoint] = 0

	return metric
}

// Update the metric values
func (m *ErrorRatePerEndpoint) Update(params map[string]interface{}) error {
	endpointName := m.endpointName(params)
	m.lock.Lock()
	if params["statusCode"].(int) >= 400 {
		m.errorCount[endpointName]++
	}
	m.reqCount[endpointName]++
	m.lock.Unlock()

	return nil
}

// ValueMap extract all the metrics to be reported
func (m *ErrorRatePerEndpoint) ValueMap() map[string]float32 {

	metrics := make(map[string]float32)

	m.lock.Lock()
	var allEPErrors int
	var reqAllEndpoints int
	for endpoint := range m.errorCount {
		metricName := m.namePrefix + endpoint + m.metricUnit

		metrics[metricName] = 0.
		if overallReq := float32(m.reqCount[endpoint]); overallReq > 0.0 {
			metrics[metricName] = float32(m.errorCount[endpoint]) / overallReq
		}

		allEPErrors += m.errorCount[endpoint]
		reqAllEndpoints += m.reqCount[endpoint]

		m.errorCount[endpoint] = 0
		m.reqCount[endpoint] = 0
	}

	metrics[m.allEPNamePrefix+m.metricUnit] = 0.
	if reqAllEndpoints > 0 {
		metrics[m.allEPNamePrefix+m.metricUnit] = float32(allEPErrors) / float32(reqAllEndpoints)
	}

	m.lock.Unlock()

	return metrics
}

/**************************************************
* Response time per endpoint
**************************************************/

// ResponseTimePerEndpoint tracks the response time per endpoint
type ResponseTimePerEndpoint struct {
	*StandardMetric
	responseTimeMap map[string][]float32
}

// NewResponseTimePerEndpoint creates new ResponseTimePerEndpoint metric
func NewResponseTimePerEndpoint() *ResponseTimePerEndpoint {

	metric := &ResponseTimePerEndpoint{
		StandardMetric: &StandardMetric{
			reqCount:        make(map[string]int),
			namePrefix:      "Component/ResponseTimePerEndpoint/",
			allEPNamePrefix: "Component/ResponseTime/overall",
			metricUnit:      "[ms]",
		},

		responseTimeMap: make(map[string][]float32),
	}

	// initialize the metrics
	metric.initReqCount()
	for endpoint := range metric.endpoints {
		metric.responseTimeMap[endpoint] = make([]float32, 1)
	}
	metric.responseTimeMap[unknownEndpoint] = make([]float32, 1)

	return metric
}

// Update the metric values
func (m *ResponseTimePerEndpoint) Update(params map[string]interface{}) error {

	startTime, ok := params["reqStartTime"]
	if !ok {
		return errors.New("reqStart time should be time.Time")
	}

	elaspsedTimeInMs := float32(time.Since(startTime.(time.Time))) / float32(time.Millisecond)

	endpointName := m.endpointName(params)
	m.lock.Lock()
	m.reqCount[endpointName]++
	m.responseTimeMap[endpointName] = append(m.responseTimeMap[endpointName], elaspsedTimeInMs)
	m.lock.Unlock()

	return nil
}

// ValueMap extract all the metrics to be reported
func (m *ResponseTimePerEndpoint) ValueMap() map[string]float32 {

	metrics := make(map[string]float32)

	m.lock.Lock()
	defer m.lock.Unlock()

	var responseTimeAllEndpoints float32
	var numReqAllEndpoints int

	for endpoint, values := range m.responseTimeMap {

		var responseTimeSum float32
		for _, value := range values {
			responseTimeSum += value
		}

		metricName := m.namePrefix + endpoint + m.metricUnit
		metrics[metricName] = 0.

		if numReq := float32(m.reqCount[endpoint]); numReq > 0 {
			metrics[metricName] = float32(responseTimeSum) / numReq
		}

		responseTimeAllEndpoints += responseTimeSum
		numReqAllEndpoints += m.reqCount[endpoint]

		m.reqCount[endpoint] = 0
		m.responseTimeMap[endpoint] = make([]float32, 1)
	}

	metrics[m.allEPNamePrefix+m.metricUnit] = 0.
	if numReqAllEndpoints > 0 {
		metrics[m.allEPNamePrefix+m.metricUnit] = responseTimeAllEndpoints / float32(numReqAllEndpoints)
	}

	return metrics
}
