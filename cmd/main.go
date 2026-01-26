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

	secretctrl "github.com/kyma-project/infrastructure-manager/internal/controller/secret"
	"github.com/kyma-project/infrastructure-manager/internal/rtbootstrapper"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenerapis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/go-logr/logr"
	validator "github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	infrastructuremanagerv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	kubeconfigcontroller "github.com/kyma-project/infrastructure-manager/internal/controller/kubeconfig"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics"
	registrycachecontroller "github.com/kyma-project/infrastructure-manager/internal/controller/registrycache"
	runtimecontroller "github.com/kyma-project/infrastructure-manager/internal/controller/runtime"
	"github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/token"
)

var (
	scheme   = runtime.NewScheme()        //nolint:gochecknoglobals
	setupLog = ctrl.Log.WithName("setup") //nolint:gochecknoglobals
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(infrastructuremanagerv1.AddToScheme(scheme))
	utilruntime.Must(rbacv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

// Default values for the Runtime controller configuration
const (
	defaultControlPlaneRequeueDuration   = 10 * time.Second
	defaultGardenerRequestTimeout        = 3 * time.Second
	defaultGardenerRateLimiterQPS        = 5
	defaultGardenerRateLimiterBurst      = 5
	defaultMinimalRotationTimeRatio      = 0.6
	defaultExpirationTime                = 24 * time.Hour
	defaultGardenerReconciliationTimeout = 60 * time.Second
	defaultGardenerRequeueDuration       = 15 * time.Second
	defaultShootCreateRequeueDuration    = 60 * time.Second
	defaultShootDeleteRequeueDuration    = 90 * time.Second
	defaultShootReconcileRequeueDuration = 30 * time.Second
	defaultRuntimeCtrlWorkersCnt         = 25
	defaultGardenerClusterCtrlWorkersCnt = 25
)

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var gardenerKubeconfigPath string
	var gardenerProjectName string
	var minimalRotationTimeRatio float64
	var expirationTime time.Duration
	var gardenerCtrlReconciliationTimeout time.Duration
	var runtimeCtrlGardenerRequestTimeout time.Duration
	var runtimeCtrlGardenerRateLimiterQPS int
	var runtimeCtrlGardenerRateLimiterBurst int
	var runtimeCtrlWorkersCnt int
	var gardenerClusterCtrlWorkersCnt int
	var converterConfigFilepath string
	var auditLogMandatory bool
	var registryCacheConfigControllerEnabled bool
	var runtimeBootstrapperEnabled bool
	var runtimeBootstrapperManifestsPath string
	var runtimeBootstrapperConfigName string
	var runtimeBootstrapperPullSecretName string
	var runtimeBootstrapperClusterTrustBundle string
	var runtimeBootstrapperDeploymentName string

	//Kubebuilder related parameters:
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to. Monitoring and alerting tools can use this endpoint to collect application specific metrics during runtime")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to. Kubernetes is using the probe endpoint to determine the health state of the application process")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	//Gardener related parameters:
	flag.StringVar(&gardenerKubeconfigPath, "gardener-kubeconfig-path", "/gardener/kubeconfig/kubeconfig", "Path to the kubeconfig file by KIM to access the for Gardener cluster")
	flag.StringVar(&gardenerProjectName, "gardener-project-name", "gardener-project", "Name of the Gardener project which is used for storing Shoot definitions")

	// Kubeconfig Controller specific parameters:
	flag.Float64Var(&minimalRotationTimeRatio, "minimal-rotation-time", defaultMinimalRotationTimeRatio, "The ratio determines what is the minimal time that needs to pass to rotate the kubeconfig of Shoot clusters. "+
		"The ratio determines what is the minimal time that needs to pass to rotate the kubeconfig of Shoot clusters. "+
		"For example if `kubeconfig-expiration-time` is set to `24hs` and `minimal-rotation-time` is set to `0.5`, then the next reconciliation after 12 hours will trigger the rotation")
	flag.DurationVar(&expirationTime, "kubeconfig-expiration-time", defaultExpirationTime, "Expiration time is the maximum age of a Shoot kubeconfig until it is considered as invalid")
	flag.DurationVar(&gardenerCtrlReconciliationTimeout, "gardener-ctrl-reconcilation-timeout", defaultGardenerReconciliationTimeout, "Timeout duration for reconiling a kubeconfig for Gardener Cluster Controller. The reconciliation of a kubeconfig is cancelled when this timeout is reached")
	flag.IntVar(&gardenerClusterCtrlWorkersCnt, "gardener-cluster-ctrl-workers-cnt", defaultGardenerClusterCtrlWorkersCnt, "Number of workers running in parallel for Gardener Cluster Controller. The number of parallel workers has an impact on the amount of requests send to the Gardener cluster")

	// Runtime Controller specific parameters:
	flag.DurationVar(&runtimeCtrlGardenerRequestTimeout, "gardener-request-timeout", defaultGardenerRequestTimeout, "Timeout duration for Gardener client for Runtime Controller. Requests to the Gardener cluster are cancelled when this timeout is reached")
	flag.IntVar(&runtimeCtrlGardenerRateLimiterQPS, "gardener-ratelimiter-qps", defaultGardenerRateLimiterQPS, "Gardener client rate limiter QPS (queries per seconds) for Runtime Controller. The queries per second has direct impact on the load produced for the Gardener cluster (see https://cloud.google.com/config-connector/docs/how-to/customize-controller-manager-rate-limit)")
	flag.IntVar(&runtimeCtrlGardenerRateLimiterBurst, "gardener-ratelimiter-burst", defaultGardenerRateLimiterBurst, "Gardener client rate limiter burst for Runtime Controller. The burst value allows for more requests than the qps limit for short periods (see https://cloud.google.com/config-connector/docs/how-to/customize-controller-manager-rate-limit)")
	flag.IntVar(&runtimeCtrlWorkersCnt, "runtime-ctrl-workers-cnt", defaultRuntimeCtrlWorkersCnt, "Number of workers running in parallel for Runtime Controller. The number of parallel workers has an impact on the amount of requests send to the Gardener cluster")
	flag.StringVar(&converterConfigFilepath, "converter-config-filepath", "/converter-config/converter_config.json", "File path to the gardener shoot converter configuration.")

	//Feature flags:
	flag.BoolVar(&auditLogMandatory, "audit-log-mandatory", true, "Feature flag to enable strict mode for audit log configuration. When enabled this feature, a Shoot cluster will only be created when an auditlog tenant exists (this is defined in the auditlog mapping configuration file)")
	flag.BoolVar(&registryCacheConfigControllerEnabled, "registry-cache-config-controller-enabled", false, "Feature flag to enable registry cache config controller")
	flag.BoolVar(&runtimeBootstrapperEnabled, "runtime-bootstrapper-enabled", false, "Feature flag to enable runtime bootstrapper")

	// Runtime bootstrapper configuration
	flag.StringVar(&runtimeBootstrapperManifestsPath, "runtime-bootstrapper-manifests-path", "/webhook/manifests.yaml", "File path to the manifests containing runtime bootstrapper.")
	flag.StringVar(&runtimeBootstrapperConfigName, "runtime-bootstrapper-config-name", "rt-bootstrapper-config", "Name of the the runtime bootstrapper Config Map.")
	flag.StringVar(&runtimeBootstrapperPullSecretName, "runtime-bootstrapper-pull-secret-name", "", "Name of the pull secret to be copied to SKR.")
	flag.StringVar(&runtimeBootstrapperClusterTrustBundle, "runtime-bootstrapper-cluster-trust-bundle", "", "Cluster trust bundle to be copied to SKR.")
	flag.StringVar(&runtimeBootstrapperDeploymentName, "runtime-bootstrapper-deployment-namespaced-name", "kyma-system/rt-bootstrapper-controller-manager", "Name of the deployment to be observed to verify if installation succeeded. Expected format: <namespace>/<name>")

	opts := zap.Options{}
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
		Cache:                  restrictWatchedNamespace(),
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
	gardenerClient, shootClient, dynamicKubeconfigClient, err := initGardenerClients(gardenerKubeconfigPath, gardenerNamespace, runtimeCtrlGardenerRequestTimeout, runtimeCtrlGardenerRateLimiterQPS, runtimeCtrlGardenerRateLimiterBurst)

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
	if err = kubeconfigcontroller.NewGardenerClusterController(
		mgr,
		kubeconfigProvider,
		logger,
		rotationPeriod,
		minimalRotationTimeRatio,
		gardenerCtrlReconciliationTimeout,
		metrics,
	).SetupWithManager(mgr, gardenerClusterCtrlWorkersCnt); err != nil {
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

	if runtimeBootstrapperEnabled && runtimeBootstrapperClusterTrustBundle != "" {
		// ClusterTrustBundle is a beta feature and needs to be explicitly enabled in the converter config
		// When the feature is generally available, this code can be removed
		// As of the time of writing (December 2025) there is no GA release date announced
		// Details: https://kubernetes.io/docs/reference/access-authn-authz/certificate-signing-requests/#cluster-trust-bundles
		enableClusterTrustBundleFeatureForSKR(&config.ConverterConfig)
	}

	auditLogDataMap, err := loadAuditLogDataMap(config.ConverterConfig.AuditLog.TenantConfigPath)
	if err != nil {
		setupLog.Error(err, "invalid audit log tenant configuration")
		os.Exit(1)
	}

	_, err = token.ValidateTokenExpirationTime(config.ConverterConfig.Kubernetes.KubeApiServer.MaxTokenExpiration)
	if err != nil {
		setupLog.Error(err, "invalid token expiration format in converter configuration")
		os.Exit(1)
	}

	runtimeClientGetter := fsm.NewRuntimeClientGetter(mgr.GetClient())

	var runtimeBootstrapperInstaller *rtbootstrapper.Installer

	if runtimeBootstrapperEnabled {

		rtbConfig := rtbootstrapper.Config{
			PullSecretName:           runtimeBootstrapperPullSecretName,
			ClusterTrustBundleName:   runtimeBootstrapperClusterTrustBundle,
			ManifestsPath:            runtimeBootstrapperManifestsPath,
			ConfigName:               runtimeBootstrapperConfigName,
			DeploymentNamespacedName: runtimeBootstrapperDeploymentName,
		}

		runtimeBootstrapperInstaller, err = configureRuntimeBootstrapper(rtbConfig)
		if err != nil {
			setupLog.Error(err, "unable to initialize runtime bootstrapper installer")
			os.Exit(1)
		}
	}

	cfg := fsm.RCCfg{
		GardenerRequeueDuration:              defaultGardenerRequeueDuration,
		RequeueDurationShootCreate:           defaultShootCreateRequeueDuration,
		RequeueDurationShootDelete:           defaultShootDeleteRequeueDuration,
		RequeueDurationShootReconcile:        defaultShootReconcileRequeueDuration,
		ControlPlaneRequeueDuration:          defaultControlPlaneRequeueDuration,
		Finalizer:                            infrastructuremanagerv1.Finalizer,
		ShootNamesapace:                      gardenerNamespace,
		Config:                               config,
		AuditLogMandatory:                    auditLogMandatory,
		Metrics:                              metrics,
		AuditLogging:                         auditLogDataMap,
		RegistryCacheConfigControllerEnabled: registryCacheConfigControllerEnabled,
		RuntimeBootstrapperEnabled:           runtimeBootstrapperEnabled,
		RuntimeBootstrapperInstaller:         runtimeBootstrapperInstaller,
	}

	runtimeReconciler := runtimecontroller.NewRuntimeReconciler(
		mgr,
		gardenerClient,
		runtimeClientGetter,
		runtimeBootstrapperInstaller,
		logger,
		cfg,
	)

	if err = runtimeReconciler.SetupWithManager(mgr, runtimeCtrlWorkersCnt); err != nil {
		setupLog.Error(err, "unable to setup controller with Manager", "controller", "Runtime")
		os.Exit(1)
	}

	if err := (&secretctrl.SecretReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Secret")
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

	if registryCacheConfigControllerEnabled {
		registryCacheConfigReconciler := registrycachecontroller.NewRegistryCacheConfigReconciler(mgr, logger, gardener.GetRuntimeClient)
		if err = registryCacheConfigReconciler.SetupWithManager(mgr, 1); err != nil {
			setupLog.Error(err, "unable to setup registry cache config controller with Manager", "controller", "Runtime")
			os.Exit(1)
		}
	}

	setupLog.Info("Starting Manager", "kubeconfigExpirationTime", expirationTime, "kubeconfigRotationPeriod", rotationPeriod)

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func initGardenerClients(kubeconfigPath string, namespace string, timeout time.Duration, rlQPS, rlBurst int) (client.Client, gardenerapis.ShootInterface, client.SubResourceClient, error) {
	restConfig, err := gardener.NewRestConfigFromFile(kubeconfigPath)
	if err != nil {
		return nil, nil, nil, err
	}

	restConfig.Timeout = timeout
	restConfig.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(float32(rlQPS), rlBurst)

	gardenerClientSet, err := gardenerapis.NewForConfig(restConfig)
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

func restrictWatchedNamespace() cache.Options {
	return cache.Options{
		ByObject: map[client.Object]cache.ByObject{
			&corev1.Secret{}: {
				Label: k8slabels.Everything(),
				Namespaces: map[string]cache.Config{
					"kcp-system": {},
				},
			},
			&infrastructuremanagerv1.Runtime{}: {
				Namespaces: map[string]cache.Config{
					"kcp-system": {},
				},
			},
			&infrastructuremanagerv1.GardenerCluster{}: {
				Namespaces: map[string]cache.Config{
					"kcp-system": {},
				},
			},
		},
	}
}

func configureRuntimeBootstrapper(config rtbootstrapper.Config) (*rtbootstrapper.Installer, error) {
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to verify Runtime Bootstrapper configuration")
	}

	kcpClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, errors.Wrap(err, "failed to verify Runtime Bootstrapper configuration")
	}

	err = rtbootstrapper.NewValidator(config, kcpClient).Validate(context.Background())
	if err != nil {
		return nil, err
	}

	return rtbootstrapper.NewInstaller(config, kcpClient, fsm.NewRuntimeClientGetter(kcpClient), fsm.NewRuntimeDynamicClientGetter(kcpClient)), nil
}

func enableClusterTrustBundleFeatureForSKR(converterConfig *config.ConverterConfig) {
	if converterConfig.Kubernetes.KubeApiServer.FeatureGates == nil {
		converterConfig.Kubernetes.KubeApiServer.FeatureGates = make(map[string]bool)
	}

	if converterConfig.Kubernetes.KubeApiServer.RuntimeConfig == nil {
		converterConfig.Kubernetes.KubeApiServer.RuntimeConfig = make(map[string]bool)
	}

	// Feature gates docs: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
	converterConfig.Kubernetes.KubeApiServer.FeatureGates["ClusterTrustBundle"] = true
	// Runtime Bootstrapper requires ClusterTrustBundleProjection to be enabled as well to mount the trust bundle into pods
	converterConfig.Kubernetes.KubeApiServer.FeatureGates["ClusterTrustBundleProjection"] = true
	converterConfig.Kubernetes.KubeApiServer.RuntimeConfig["certificates.k8s.io/v1beta1/clustertrustbundles"] = true

	if converterConfig.Kubernetes.Kubelet.FeatureGates == nil {
		converterConfig.Kubernetes.Kubelet.FeatureGates = make(map[string]bool)
	}
	converterConfig.Kubernetes.Kubelet.FeatureGates["ClusterTrustBundleProjection"] = true
}
