/*


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
	"github.com/vmware/govmomi/session/cache"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
	"net/url"

	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	topologyv1 "viminfo/api/v1"
	"viminfo/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = topologyv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

// - vSphere session login function

func vlogin(ctx context.Context, vc, user, pwd string) (*vim25.Client, error) {

	//
	// Create a vSphere/vCenter client
	//
	// The govmomi client requires a URL object, u.
	// You cannot use a string representation of the vCenter URL.
	// soap.ParseURL provides the correct object format.
	//

	u, err := soap.ParseURL(vc)

	if u == nil {
		setupLog.Error(err, "Unable to parse URL. Are required environment variables set?", "controller", "VMInfo")
		os.Exit(1)
	}

	if err != nil {
		setupLog.Error(err, "URL parsing not successful", "controller", "VMInfo")
		os.Exit(1)
	}

	u.User = url.UserPassword(user, pwd)

	//
	// Session cache example taken from https://github.com/vmware/govmomi/blob/master/examples/examples.go
	//
	// Share govc's session cache
	//
	s := &cache.Session{
		URL:      u,
		Insecure: true,
	}

	//
	// Create new client
	//
	c := new(vim25.Client)

	//
	// Login using client c and cache s
	//
	err = s.Login(ctx, c, nil)

	if err != nil {
		setupLog.Error(err, " login not successful", "controller", "VMInfo")
		os.Exit(1)
	}

	return c, nil
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "517fb653.corinternal.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	//
	// Retrieve vCenter URL, username and password from environment variables
	// These are provided via the manager manifest when controller is deployed
	//

	vc := os.Getenv("GOVMOMI_URL")
	user := os.Getenv("GOVMOMI_USERNAME")
	pwd := os.Getenv("GOVMOMI_PASSWORD")

	//
	// Create context, and get vSphere session information
	//

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, err := vlogin(ctx, vc, user, pwd)
	if err != nil {
		setupLog.Error(err, "unable to get login session to vSphere")
		os.Exit(1)
	}

	//
	// Add a new field, VC, to send session info to Reconciler
	//
	if err = (&controllers.VMInfoReconciler{
		Client: mgr.GetClient(),
		VC:     c,
		Log:    ctrl.Log.WithName("controllers").WithName("VMInfo"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VMInfo")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
