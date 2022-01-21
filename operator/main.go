/*
Copyright 2020 In Loco Tecnologia da Informação S.A.

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
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"time"

	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	inlocov1alpha1 "github.com/inloco/kube-actions/operator/api/v1alpha1"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunner"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunnerjob"
	"github.com/inloco/kube-actions/operator/controllers/actionsrunnerreplicaset"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = apimachineryruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(inlocov1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	go http.ListenAndServe(":6060", nil)

	var maxConcurrentReconciles int
	flag.IntVar(&maxConcurrentReconciles,
		"max-concurrent-reconciles",
		runtime.NumCPU(),
		"Maximum number of concurrent reconciles for controllers.",
	)

	var metricsBindAddress string
	flag.StringVar(&metricsBindAddress,
		"metrics-bind-address",
		":8080",
		"The address the metric endpoint binds to.",
	)

	var healthProbeBindAddress string
	flag.StringVar(&healthProbeBindAddress,
		"health-probe-bind-address",
		":8081",
		"The address the probe endpoint binds to.",
	)

	var leaderElect bool
	flag.BoolVar(
		&leaderElect,
		"leader-elect",
		false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.",
	)

	var leaderElectionNamespace string
	flag.StringVar(
		&leaderElectionNamespace,
		"leader-election-namespace",
		"kube-actions",
		"Namespace in which to create the leader election configmap for holding the leader lock (required if running locally with leader election enabled).",
	)

	var leaderElectionId string
	flag.StringVar(
		&leaderElectionId,
		"leader-election-id",
		"00000000.inloco.com.br",
		"Name of the configmap that is used for holding the leader lock.",
	)

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)

	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(
		ctrl.GetConfigOrDie(),
		ctrl.Options{
			Scheme:                  scheme,
			MetricsBindAddress:      metricsBindAddress,
			HealthProbeBindAddress:  healthProbeBindAddress,
			LeaderElection:          leaderElect,
			LeaderElectionNamespace: leaderElectionNamespace,
			LeaderElectionID:        leaderElectionId,
			Port:                    9443,
		},
	)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	var arrs inlocov1alpha1.ActionsRunnerReplicaSet
	if err := arrs.SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "ActionsRunnerReplicaSet")
		os.Exit(1)
	}

	var ar inlocov1alpha1.ActionsRunner
	if err := ar.SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "ActionsRunner")
		os.Exit(1)
	}

	arrsReconciler := actionsrunnerreplicaset.Reconciler{
		Client:                  mgr.GetClient(),
		Log:                     mgr.GetLogger(),
		Scheme:                  mgr.GetScheme(),
		MaxConcurrentReconciles: maxConcurrentReconciles,
	}
	if err := arrsReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ActionsRunnerReplicaSet")
		os.Exit(1)
	}

	arReconciler := actionsrunner.Reconciler{
		Client:                  mgr.GetClient(),
		Log:                     mgr.GetLogger(),
		Scheme:                  mgr.GetScheme(),
		MaxConcurrentReconciles: maxConcurrentReconciles,
	}
	if err := arReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "actionsRunner")
		os.Exit(1)
	}

	arjReconciler := actionsrunnerjob.Reconciler{
		Client:                  mgr.GetClient(),
		Scheme:                  mgr.GetScheme(),
		MaxConcurrentReconciles: maxConcurrentReconciles,
	}
	if err := arjReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ActionsRunnerJob")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

	time.Sleep(time.Minute)
}
