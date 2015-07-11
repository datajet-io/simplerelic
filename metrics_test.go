package simplerelic

import "testing"

const (
	endpointName = "log"
)

func TestReqPerEndpoint(t *testing.T) {

	m := NewReqPerEndpoint()

	params := make(map[string]interface{})
	params["endpointName"] = "log"
	m.Update(params)

	// we make a request and retrieve the metric's values right after
	_ = m.ValueMap()

	// We make another request and retrieve the metric's values.
	// We're supposed to capture both requests as there was no call to Clear
	// function just yet. This behaviour is useful in case the sending
	// to NewRelic fails and we need to aggregate the previously existing
	// metric values.
	m.Update(params)
	values := m.ValueMap()

	m.Clear()

	// check the error rate calculation
	metricName := "Component/ReqPerEndpoint/log[requests]"
	if values[metricName] != 2 {
		t.Errorf("expected 2, got %f", values[metricName])
	}
}

func TestErrorRate(t *testing.T) {

	m := NewErrorRatePerEndpoint()

	params := make(map[string]interface{})
	params["endpointName"] = "log"

	params["statusCode"] = 404
	m.Update(params)

	params["statusCode"] = 200
	m.Update(params)

	_ = m.ValueMap()

	params["statusCode"] = 200
	m.Update(params)
	m.Update(params)

	values := m.ValueMap()

	metricName := "Component/ErrorRatePerEndpoint/log[percent]"
	if values[metricName] != 0.25 {
		t.Errorf("expected 0.25, got %f", values[metricName])
	}
}

/*func TestResponseTimeValueMap(t *testing.T) {

	setup()

	m := NewResponseTimePerEndpoint()

	engine.GET("/log", func(c *gin.Context) {

		ts := []float32{0.1, 0.2, 0.1, 0.2}
		for _, t := range ts {
			m.responseTimeMap[endpointName] = append(m.responseTimeMap[endpointName], t)
			m.reqCount[endpointName]++
		}
	})

	engine.ServeHTTP(recorder, req)

	values := m.ValueMap()
}*/
