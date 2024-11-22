/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	gardener_oidc "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	"github.com/go-logr/logr"
	validator "github.com/go-playground/validator/v10"
	infrastructuremanagerv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	kubeconfig_controller "github.com/kyma-project/infrastructure-manager/internal/controller/kubeconfig"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics"
	runtime_controller "github.com/kyma-project/infrastructure-manager/internal/controller/runtime"
	"github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	scheme   = runtime.NewScheme()        //nolint:gochecknoglobals
	setupLog = ctrl.Log.WithName("setup") //nolint:gochecknoglobals
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(infrastructuremanagerv1.AddToScheme(scheme))
	utilruntime.Must(rbacv1.AddToScheme(scheme))
	utilruntime.Must(gardener_oidc.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

const defaultMinimalRotationTimeRatio = 0.6
const defaultExpirationTime = 24 * time.Hour
const defaultGardenerRequestTimeout = 60 * time.Second
const defaultControlPlaneRequeueDuration = 10 * time.Second
const defaultGardenerRateLimiterQps = 5
const defaultGardenerRateLimiterBurst = 5

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var gardenerKubeconfigPath string
	var gardenerProjectName string
	var minimalRotationTimeRatio float64
	var expirationTime time.Duration
	var gardenerRequestTimeout time.Duration
	var runtimeControllerGardenerRequestTimeout time.Duration
	var runtimeControllerGardenerRateLimiterQps int
	var runtimeControllerGardenerRateLimiterBurst int
	var converterConfigFilepath string
	var shootSpecDumpEnabled bool
	var auditLogMandatory bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&gardenerKubeconfigPath, "gardener-kubeconfig-path", "/gardener/kubeconfig/kubeconfig", "Kubeconfig file for Gardener cluster")
	flag.StringVar(&gardenerProjectName, "gardener-project-name", "gardener-project", "Name of the Gardener project")
	flag.Float64Var(&minimalRotationTimeRatio, "minimal-rotation-time", defaultMinimalRotationTimeRatio, "The ratio determines what is the minimal time that needs to pass to rotate certificate.")
	flag.DurationVar(&expirationTime, "kubeconfig-expiration-time", defaultExpirationTime, "Dynamic kubeconfig expiration time")
	flag.DurationVar(&gardenerRequestTimeout, "gardener-request-timeout", defaultGardenerRequestTimeout, "Timeout duration for requests to Gardener")
	flag.DurationVar(&runtimeControllerGardenerRequestTimeout, "rt-ctrl-gardener-request-timeout", defaultGardenerRequestTimeout, "Timeout duration for requests from Runtime Controller to Gardener")
	flag.IntVar(&runtimeControllerGardenerRateLimiterQps, "rt-ctrl-gardener-ratelimiter-qps", defaultGardenerRateLimiterQps, "Timeout duration for requests to Gardener")
	flag.IntVar(&runtimeControllerGardenerRateLimiterBurst, "rt-ctrl-gardener-ratelimiter-burst", defaultGardenerRateLimiterBurst, "Timeout duration for requests to Gardener")
	flag.StringVar(&converterConfigFilepath, "converter-config-filepath", "/converter-config/converter_config.json", "A file path to the gardener shoot converter configuration.")
	flag.BoolVar(&shootSpecDumpEnabled, "shoot-spec-dump-enabled", false, "Feature flag to allow persisting specs of created shoots")
	flag.BoolVar(&auditLogMandatory, "audit-log-mandatory", true, "Feature flag to enable strict mode for audit log configuration")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)

	restConfig := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},

		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "f1c68560.kyma-project.io",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	gardenerNamespace := fmt.Sprintf("garden-%s", gardenerProjectName)
	gardenerClient, shootClient, dynamicKubeconfigClient, err := initGardenerClients(gardenerKubeconfigPath, gardenerNamespace)

	if err != nil {
		setupLog.Error(err, "unable to initialize gardener clients", "controller", "GardenerCluster")
		os.Exit(1)
	}

	kubeconfigProvider := kubeconfig.NewKubeconfigProvider(
		shootClient,
		dynamicKubeconfigClient,
		gardenerNamespace,
		int64(expirationTime.Seconds()))

	rotationPeriod := time.Duration(minimalRotationTimeRatio*expirationTime.Minutes()) * time.Minute
	metrics := metrics.NewMetrics()
	if err = kubeconfig_controller.NewGardenerClusterController(
		mgr,
		kubeconfigProvider,
		logger,
		rotationPeriod,
		minimalRotationTimeRatio,
		gardenerRequestTimeout,
		metrics,
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GardenerCluster")
		os.Exit(1)
	}

	// load converter configuration
	getReader := func() (io.Reader, error) {
		return os.Open(converterConfigFilepath)
	}
	var config config.Config
	if err = config.Load(getReader); err != nil {
		setupLog.Error(err, "unable to load converter configuration")
		os.Exit(1)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err = validate.Struct(config); err != nil {
		setupLog.Error(err, "invalid converter configuration")
		os.Exit(1)
	}

	auditLogDataMap, err := loadAuditLogDataMap(config.ConverterConfig.AuditLog.TenantConfigPath)
	if err != nil {
		setupLog.Error(err, "invalid audit log tenant configuration")
		os.Exit(1)
	}

	cfg := fsm.RCCfg{
		GardenerRequeueDuration:     defaultGardenerRequeueDuration,
		ControlPlaneRequeueDuration: defaultControlPlaneRequeueDuration,
		Finalizer:                   infrastructuremanagerv1.Finalizer,
		ShootNamesapace:             gardenerNamespace,
		Config:                      config,
		AuditLogMandatory:           auditLogMandatory,
		Metrics:                     metrics,
		AuditLogging:                auditLogDataMap,
	}
	if shootSpecDumpEnabled {
		cfg.PVCPath = "/testdata/kim"
	}

	runtimeReconciler := runtime_controller.NewRuntimeReconciler(
		mgr,
		gardenerClient,
		logger,
		cfg,
	)

	if err = runtimeReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to setup controller with Manager", "controller", "Runtime")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	refreshRuntimeMetrics(restConfig, logger, metrics)

	setupLog.Info("Starting Manager", "kubeconfigExpirationTime", expirationTime, "kubeconfigRotationPeriod", rotationPeriod)

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func initGardenerClients(kubeconfigPath string, namespace string) (client.Client, gardener_apis.ShootInterface, client.SubResourceClient, error) {
	restConfig, err := gardener.NewRestConfigFromFile(kubeconfigPath)
	if err != nil {
		return nil, nil, nil, err
	}

	restConfig.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(5, 5)

	gardenerClientSet, err := gardener_apis.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, nil, err
	}

	gardenerClient, err := client.New(restConfig, client.Options{})
	if err != nil {
		return nil, nil, nil, err
	}

	err = v1beta1.AddToScheme(gardenerClient.Scheme())
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to register Gardener schema")
	}

	err = gardener_oidc.AddToScheme(gardenerClient.Scheme())
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to register Gardener schema")
	}

	shootClient := gardenerClientSet.Shoots(namespace)
	dynamicKubeconfigAPI := gardenerClient.SubResource("adminkubeconfig")

	return gardenerClient, shootClient, dynamicKubeconfigAPI, nil
}

func loadAuditLogDataMap(p string) (auditlogs.Configuration, error) {
	file, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	var data auditlogs.Configuration
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, err
	}
	validate := validator.New(validator.WithRequiredStructEnabled())

	for _, nestedMap := range data {
		for _, auditLogData := range nestedMap {
			if err := validate.Struct(auditLogData); err != nil {
				return nil, err
			}
		}
	}

	return data, nil
}

func refreshRuntimeMetrics(restConfig *rest.Config, logger logr.Logger, metrics metrics.Metrics) {
	k8sClient, err := client.New(restConfig, client.Options{})
	if err != nil {
		setupLog.Error(err, "Unable to set up client for refreshing runtime CR metrics")
		os.Exit(1)
	}

	err = infrastructuremanagerv1.AddToScheme(k8sClient.Scheme())
	if err != nil {
		setupLog.Error(err, "unable to set up client")
		os.Exit(1)
	}

	logger.Info("Refreshing runtime CR metrics")
	metrics.ResetRuntimeMetrics()
	rl := infrastructuremanagerv1.RuntimeList{}
	if err = k8sClient.List(context.Background(), &rl, &client.ListOptions{Namespace: "kcp-system"}); err != nil {
		setupLog.Error(err, "error while listing unable to list runtimes")
		os.Exit(1)
	}

	for _, rt := range rl.Items {
		metrics.SetRuntimeStates(rt)
	}
}
