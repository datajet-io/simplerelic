# SimpleRelic

SimpleRelic is a Go reporting library sending http metrics to NewRelic. It is designed with simplicity in mind i.e. to add a new metric you only have to implement a struct with two functions. There are currently three default
metrics.

Default metrics:
- number of requests per endpoint
- percentage of 4xx and 5xx errors per endpoint
- response time per endpoint

The library is consisting of two parts:
- sending metrics to NewRelic Plugin API
- updating the HTTP metrics (on every request)

Metrics will be collected and sent to NewRelic every 60 seconds by a separate go routine. Updating the HTTP metrics requires that you wrap your request handler with a function that updates the metrics (see example below).

Ideally you will create your own dashboards and graphs in NewRelic (see the Custom NewRelic Plugin at the bottom of the page). Example of a dashboard you can see here.

[Dashboard](http://imgur.com/1O2lfqb)

## Basic usage

Create a default reporter that uses default metrics.

```
reporter, err := simplerelic.InitDefaultReporter(cfg.NewRelicName, cfg.NewRelicKey, cfg.DebugMode)
if err != nil {
    // handle error
}
reporter.Start()
```

The code above does the initialisation of the reporter. In order to track and update the http metrics, you need to wrap your http request handler function with a function that updates the metrics. In case you're using Gin framework you can use the snippet below,
otherwise adopt it accordingly.

```
func makeHandler(fn func(*gin.Context), endpointName string) gin.HandlerFunc {

	if simplerelic.Engine == nil {
		// handle error
	}

    return func(c *gin.Context) {
		params := simplerelic.DefaultReqParams(endpointName)
		fn(c)
		simplerelic.CollectParamsOnReqEnd(params, c.Writer.Status())
		simplerelic.UpdateMetricsOnReqEnd(params)
	}
}
```

In the example above parameter fn is your original handler. Parameter endpointName is required by default metrics to identify the
endpoint you are reporting the values for. Metrics can take additional parameters passed in a params variable (map[string]interface{}).
The parameters for default metrics are mostly set by DefaultReqParams function except for `statusCode` that needs to be set later
in the request lifetime. The metric values are updated by UpdateMetricsOnReqEnd function.

## Add an user defined metric

User defined metrics need to implement AppMetric interface.

```
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
```

For example of a metric take a look at ReqPerEndpoint in metrics.go.

After you define your new metric you need to add it to the reporter.

```
reporter, err := simplerelic.InitDefaultReporter(cfg.NewRelicName, cfg.NewRelicKey, cfg.DebugMode)
if err != nil {
    // handle error
}
reporter.AddMetrics(NewUserDefinedMetric())
```

## Custom NewRelic plugin

In case you add your own metrics and want to build dashboards and graphs for them,
you need to create your own NewRelic plugin. To report metrics to your own plugin
you just need to set the GUID (before creating the reporter).

```
simplerelic.Guid = "com.example.simplerelic"
```
