package simplerelic

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiangjin/gin"
)

const (
	endpointName = "log"
)

var (
	req      *http.Request
	recorder *httptest.ResponseRecorder
	engine   *gin.Engine
)

func setup() {
	req, _ = http.NewRequest("GET", "/log", nil)
	recorder = httptest.NewRecorder()

	engine = gin.New()
}

func checkCalc(t *testing.T, values map[string]float32, expected float32) {
	for name, value := range values {
		if strings.HasSuffix(name, endpointName+"[percent]") {
			if value != expected {
				t.Errorf("error: expected %f, got %f", expected, value)
			}
		}
		if strings.HasSuffix(name, "overall[percent]") {
			if value != expected {
				t.Errorf("error: expected %f, got %f", expected, value)
			}
		}
	}
}

func checkIsCleared(t *testing.T, m AppMetric) {
	// check if the metrics are cleared
	for _, value := range m.ValueMap() {
		if value != 0. {
			t.Errorf("error: expected %f, got %f", 0., value)
		}
	}
}

func TestReqPerEndpoint(t *testing.T) {

	setup()

	m := NewReqPerEndpoint()

	// emulate a request to /log endpoint
	engine.GET("/log", func(c *gin.Context) {
		params := make(map[string]interface{})
		params["endpointName"] = "log"
		m.Update(params)
	})

	// we make a request and retrieve the metric's values right after
	engine.ServeHTTP(recorder, req)
	_ = m.ValueMap()

	// We make another request and retrieve the metric's values.
	// We're supposed to capture both requests as there was no call to Clear
	// function just yet. This behaviour is useful in case the sending
	// to NewRelic fails and we need to aggregate the previously existing
	// metric values.
	engine.ServeHTTP(recorder, req)
	values := m.ValueMap()

	m.Clear()

	// check the error rate calculation
	checkCalc(t, values, 2)
	checkIsCleared(t, m)

}

/*func TestErrorRate(t *testing.T) {

	setup()

	m := NewErrorRatePerEndpoint()

	r.GET("/log", func(c *gin.Context) {

		params := make(map[string]interface{})
		params["urlPath"] = c.Request.URL.Path

		for i := 0; i < 4; i++ {
			params["statusCode"] = 404
			m.Update(params)
		}
		for i := 0; i < 4; i++ {
			params["statusCode"] = 200
			m.Update(params)
		}
	})

	r.ServeHTTP(recorder, req)

	values := m.ValueMap()

	// check the error rate calculation
	checkCalc(t, values, 0.5)
	checkIsCleared(t, m)
}

func TestResponseTimeValueMap(t *testing.T) {

	setup()

	m := NewResponseTimePerEndpoint()

	r.GET("/log", func(c *gin.Context) {

		ts := []float32{0.1, 0.2, 0.1, 0.2}
		for _, t := range ts {
			m.responseTimeMap[endpointName] = append(m.responseTimeMap[endpointName], t)
			m.reqCount[endpointName]++
		}
	})

	r.ServeHTTP(recorder, req)

	values := m.ValueMap()

	// check the response time calculation
	checkCalc(t, values, 0.15)
	checkIsCleared(t, m)
}*/
