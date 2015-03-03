package simplerelic

import (
	"time"
)

var (
	// Engine reports metrics to NewRelic
	Engine *Reporter
)

// InitDefaultReporter creates a new reporter and adds standard metrics
func InitDefaultReporter(appname string, licence string, verbose bool) (*Reporter, error) {

	var err error
	Engine, err = NewReporter(appname, licence, verbose)
	if err != nil {
		return nil, err
	}

	Engine.AddMetric(NewReqPerEndpoint())
	Engine.AddMetric(NewErrorRatePerEndpoint())
	Engine.AddMetric(NewResponseTimePerEndpoint())

	return Engine, nil
}

// DefaultReqParams creates and populates request parameters map to be used by default metrics
// Called in the beginning of each request
func DefaultReqParams(endpointName string) map[string]interface{} {
	params := make(map[string]interface{})
	params["endpointName"] = endpointName

	// required by response time metric
	params["reqStartTime"] = time.Now()

	return params
}

// CollectParamsOnReqEnd populates params map with additional data available when the request
// processing is already done e.g. http response status code
func CollectParamsOnReqEnd(params map[string]interface{}, statusCode int) map[string]interface{} {
	// required by error rate metric
	params["statusCode"] = statusCode
	return params
}

// UpdateMetricsOnReqEnd updates all defined metrics in the end of each request
func UpdateMetricsOnReqEnd(params map[string]interface{}) {
	for _, v := range Engine.Metrics {
		v.Update(params)
	}
}
