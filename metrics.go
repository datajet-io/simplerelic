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

	// Clear resets all the metric values
	Clear()
}

const (
	unknownEndpoint = "other"
)

// MetricBase is a base for metrics dealing with endpoints
type MetricBase struct {
	endpoints         map[string]func(urlPath string) bool
	lock              sync.RWMutex
	namePrefix        string
	namePrefixOverall string
	metricUnit        string
}

func (m *MetricBase) endpointName(params map[string]interface{}) string {
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
	*MetricBase
	numReq    map[string]int
	snapshots []*ReqPerEndpointSnapshot
}

// ReqPerEndpointSnapshot is a struct containing snapshot of the metric values
type ReqPerEndpointSnapshot struct {
	numReq map[string]int
}

// NewReqPerEndpoint creates new ReqPerEndpoint metric
func NewReqPerEndpoint() *ReqPerEndpoint {

	metric := &ReqPerEndpoint{
		MetricBase: &MetricBase{
			namePrefix:        "Component/ReqPerEndpoint/",
			namePrefixOverall: "Component/Req/overall",
			metricUnit:        "[requests]",
		},
		numReq: make(map[string]int),
	}

	// initialize the metrics
	for endpoint := range metric.endpoints {
		metric.numReq[endpoint] = 0
	}
	metric.numReq[unknownEndpoint] = 0

	return metric
}

// Update the metric values
func (m *ReqPerEndpoint) Update(params map[string]interface{}) error {
	endpointName := m.endpointName(params)
	m.lock.Lock()
	m.numReq[endpointName]++
	m.lock.Unlock()

	return nil
}

func (m *ReqPerEndpoint) doSnapshot() {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.snapshots = append(m.snapshots, &ReqPerEndpointSnapshot{numReq: m.numReq})
	m.numReq = make(map[string]int)
}

// ValueMap extract the metrics values to be reported
func (m *ReqPerEndpoint) ValueMap() map[string]float32 {

	m.doSnapshot()

	reqCount := make(map[string]float32)

	var numReqAllEndpoints int
	for _, snapshot := range m.snapshots {
		for endpoint, value := range snapshot.numReq {
			metricName := m.namePrefix + endpoint + m.metricUnit
			reqCount[metricName] += float32(value)

			numReqAllEndpoints += value
		}
	}

	m.numReq = make(map[string]int)

	reqCount[m.namePrefixOverall+m.metricUnit] = float32(numReqAllEndpoints)

	return reqCount
}

// Clear deletes all saved snapshot
func (m *ReqPerEndpoint) Clear() {
	m.snapshots = make([]*ReqPerEndpointSnapshot, 0)
}

/**************************************************
* Error rate per endpoint
**************************************************/

// ErrorRatePerEndpoint holds the percentage of error requests per endpoint
type ErrorRatePerEndpoint struct {
	*MetricBase
	numReq     map[string]int
	errorCount map[string]int
	snapshots  []*ErrorRatePerEndpointSnapshot
}

// ErrorRatePerEndpointSnapshot is a struct containing snapshot of the metric values
type ErrorRatePerEndpointSnapshot struct {
	numReq     map[string]int
	errorCount map[string]int
}

// NewErrorRatePerEndpoint creates a metric tracking error rates per endpoint
func NewErrorRatePerEndpoint() *ErrorRatePerEndpoint {

	metric := &ErrorRatePerEndpoint{
		MetricBase: &MetricBase{
			namePrefix:        "Component/ErrorRatePerEndpoint/",
			namePrefixOverall: "Component/ErrorRate/overall",
			metricUnit:        "[percent]",
		},
		numReq:     make(map[string]int),
		errorCount: make(map[string]int),
	}

	metric.init()

	return metric
}

func (m *ErrorRatePerEndpoint) init() {
	for endpoint := range m.endpoints {
		m.numReq[endpoint] = 0
		m.errorCount[endpoint] = 0
	}
	m.numReq[unknownEndpoint] = 0
	m.errorCount[unknownEndpoint] = 0
}

// Update the metric values
func (m *ErrorRatePerEndpoint) Update(params map[string]interface{}) error {
	endpointName := m.endpointName(params)
	m.lock.Lock()
	if params["statusCode"].(int) >= 400 {
		m.errorCount[endpointName]++
	}
	m.numReq[endpointName]++
	m.lock.Unlock()

	return nil
}

func (m *ErrorRatePerEndpoint) doSnapshot() {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.snapshots = append(m.snapshots, &ErrorRatePerEndpointSnapshot{
		numReq:     m.numReq,
		errorCount: m.errorCount,
	})

	m.init()
}

// ValueMap extract all the metrics to be reported
func (m *ErrorRatePerEndpoint) ValueMap() map[string]float32 {

	m.doSnapshot()

	errorCount := make(map[string]int)
	requestCount := make(map[string]int)

	for _, snapshot := range m.snapshots {
		for endpoint := range snapshot.errorCount {
			errorCount[endpoint] += snapshot.errorCount[endpoint]
			requestCount[endpoint] += snapshot.numReq[endpoint]
		}
	}

	errorRate := make(map[string]float32)

	var errorCountOverall int
	var requestCountOverall int

	for endpoint := range errorCount {
		metricName := m.namePrefix + endpoint + m.metricUnit

		errorRate[metricName] = 0.
		if overallReq := float32(requestCount[endpoint]); overallReq > 0.0 {
			errorRate[metricName] = float32(errorCount[endpoint]) / overallReq
		}

		errorCountOverall += errorCount[endpoint]
		requestCountOverall += requestCount[endpoint]
	}

	errorRate[m.namePrefixOverall+m.metricUnit] = 0.
	if requestCountOverall > 0 {
		errorRate[m.namePrefixOverall+m.metricUnit] = float32(errorCountOverall) / float32(requestCountOverall)
	}

	return errorRate
}

// Clear deletes all saved snapshot
func (m *ErrorRatePerEndpoint) Clear() {
	m.snapshots = make([]*ErrorRatePerEndpointSnapshot, 0)
}

/**************************************************
* Response time per endpoint
**************************************************/

// ResponseTimePerEndpoint tracks the response time per endpoint
type ResponseTimePerEndpoint struct {
	*MetricBase
	numReq          map[string]int
	responseTimeMap map[string][]float32
}

// NewResponseTimePerEndpoint creates new ResponseTimePerEndpoint metric
func NewResponseTimePerEndpoint() *ResponseTimePerEndpoint {

	metric := &ResponseTimePerEndpoint{
		MetricBase: &MetricBase{
			namePrefix:        "Component/ResponseTimePerEndpoint/",
			namePrefixOverall: "Component/ResponseTime/overall",
			metricUnit:        "[ms]",
		},
		numReq:          make(map[string]int),
		responseTimeMap: make(map[string][]float32),
	}

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
	m.numReq[endpointName]++
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

		if numReq := float32(m.numReq[endpoint]); numReq > 0 {
			metrics[metricName] = float32(responseTimeSum) / numReq
		}

		responseTimeAllEndpoints += responseTimeSum
		numReqAllEndpoints += m.numReq[endpoint]

		m.numReq[endpoint] = 0
		m.responseTimeMap[endpoint] = make([]float32, 1)
	}

	metrics[m.namePrefixOverall+m.metricUnit] = 0.
	if numReqAllEndpoints > 0 {
		metrics[m.namePrefixOverall+m.metricUnit] = responseTimeAllEndpoints / float32(numReqAllEndpoints)
	}

	return metrics
}

// Clear deletes all saved snapshot
func (m *ResponseTimePerEndpoint) Clear() {
	//m.snapshots = make([]*ReqPerEndpointSnapshot, 0)
}
