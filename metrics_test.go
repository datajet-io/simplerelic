package simplerelic

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

const (
	endpointName = "log"
)

var (
	req      *http.Request
	recorder *httptest.ResponseRecorder
	r        *gin.Engine
)

func setup() {
	req, _ = http.NewRequest("GET", "/log", nil)
	recorder = httptest.NewRecorder()

	r = gin.New()
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

func TestReq(t *testing.T) {

	setup()

	m := NewReqPerEndpoint()

	r.GET("/log", func(c *gin.Context) {
		params := make(map[string]interface{})
		params["endpointName"] = "log"
		m.Update(params)
	})

	r.ServeHTTP(recorder, req)

	values := m.ValueMap()

	// check the error rate calculation
	checkCalc(t, values, 1)
	checkIsCleared(t, m)

}

func TestErrorRate(t *testing.T) {

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
}
