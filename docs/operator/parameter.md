# KIM - Parameter

The Kyma Infrastructure Manager supports multiple command line parameters:

```
  -audit-log-mandatory
    	Feature flag to enable strict mode for audit log configuration (default true)
  -converter-config-filepath string
    	A file path to the gardener shoot converter configuration. (default "/converter-config/converter_config.json")
  -custom-config-controller-enabled
    	Feature flag to custom config controller
  -gardener-cluster-ctrl-workers-cnt int
    	A number of workers running in parallel for Gardener Cluster Controller (default 25)
  -gardener-ctrl-reconcilation-timeout duration
    	Timeout duration for reconlication for Gardener Cluster Controller (default 1m0s)
  -gardener-kubeconfig-path string
    	Kubeconfig file for Gardener cluster (default "/gardener/kubeconfig/kubeconfig")
  -gardener-project-name string
    	Name of the Gardener project (default "gardener-project")
  -gardener-ratelimiter-burst int
    	Gardener client rate limiter burst for Runtime Controller (default 5)
  -gardener-ratelimiter-qps int
    	Gardener client rate limiter QPS for Runtime Controller (default 5)
  -gardener-request-timeout duration
    	Timeout duration for Gardener client for Runtime Controller (default 3s)
  -health-probe-bind-address string
    	The address the probe endpoint binds to. (default ":8081")
  -kubeconfig string
    	Paths to a kubeconfig. Only required if out-of-cluster.
  -kubeconfig-expiration-time duration
    	Dynamic kubeconfig expiration time (default 24h0m0s)
  -leader-elect
    	Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.
  -metrics-bind-address string
    	The address the metric endpoint binds to. (default ":8080")
  -minimal-rotation-time float
    	The ratio determines what is the minimal time that needs to pass to rotate certificate. (default 0.6)
  -runtime-ctrl-workers-cnt int
    	A number of workers running in parallel for Runtime Controller (default 25)
  -structured-auth-enabled
    	Feature flag to enable structured authentication
  -zap-devel
    	Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error)
  -zap-encoder value
    	Zap log encoding (one of 'json' or 'console')
  -zap-log-level value
    	Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', or any integer value > 0 which corresponds to custom debug levels of increasing verbosity
  -zap-stacktrace-level value
    	Zap Level at and above which stacktraces are captured (one of 'info', 'error', 'panic').
  -zap-time-encoding value
    	Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano'). Defaults to 'epoch'.
```