// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repair

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"istio.io/istio/tools/istio-iptables/pkg/constants"
	"istio.io/pkg/log"
)

type controllerOptions struct {
	RepairOptions *Options `json:"repair_options"`
	RunAsDaemon   bool     `json:"run_as_daemon"`
	Enabled       bool     `json:"enabled"`
}

func init() {
	// Filter Options
	pflag.String("repair-node-name", "", "The name of the managed node (will manage all nodes if unset)")
	pflag.String(
		"repair-sidecar-annotation",
		"sidecar.istio.io/status",
		"An annotation key that indicates this pod contains an istio sidecar. All pods without this annotation will be ignored."+
			"The value of the annotation is ignored.")
	pflag.String(
		"repair-init-container-name",
		"istio-validation",
		"The name of the istio init container (will crash-loop if CNI is not configured for the pod)")
	pflag.String(
		"repair-init-container-termination-message",
		"",
		"The expected termination message for the init container when crash-looping because of CNI misconfiguration")
	pflag.Int(
		"repair-init-container-exit-code",
		constants.ValidationErrorCode,
		"Expected exit code for the init container when crash-looping because of CNI misconfiguration")

	pflag.String("repair-label-selectors", "", "A set of label selectors in label=value format that will be added to the pod list filters")
	pflag.String("repair-field-selectors", "", "A set of field selectors in label=value format that will be added to the pod list filters")

	// Repair Options
	pflag.Bool("repair-enabled", true, "Whether enable race condition repair or not.")
	pflag.Bool("repair-delete-pods", false, "Controller will delete pods")
	pflag.Bool("repair-label-pods", false, "Controller will label pods")
	pflag.Bool("repair-run-as-daemon", false, "Controller will run in a loop")
	pflag.String(
		"repair-broken-pod-label-key",
		"cni.istio.io/uninitialized",
		"The key portion of the label which will be set by the reconciler if --label-pods is true")
	pflag.String(
		"repair-broken-pod-label-value",
		"true",
		"The value portion of the label which will be set by the reconciler if --label-pods is true")

	pflag.Bool("repair-help", false, "Print usage information")
}

// Parse command line options
func parseFlags() (filters *Filters, options *controllerOptions) {
	// Parse command line flags
	pflag.Parse()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		log.Fatalf("Error parsing command line args: %+v", err)
	}

	if viper.GetBool("help") {
		pflag.Usage()
		os.Exit(0)
	}

	viper.AutomaticEnv()
	// Pull runtime args into structs
	filters = &Filters{
		InitContainerName:               viper.GetString("repair-init-container-name"),
		InitContainerTerminationMessage: viper.GetString("repair-init-container-termination-message"),
		InitContainerExitCode:           viper.GetInt("repair-init-container-exit-code"),
		SidecarAnnotation:               viper.GetString("repair-sidecar-annotation"),
		FieldSelectors:                  viper.GetString("repair-field-selectors"),
		LabelSelectors:                  viper.GetString("repair-label-selectors"),
	}
	options = &controllerOptions{
		Enabled:     viper.GetBool("repair-enabled"),
		RunAsDaemon: viper.GetBool("repair-run-as-daemon"),
		RepairOptions: &Options{
			DeletePods:    viper.GetBool("repair-delete-pods"),
			LabelPods:     viper.GetBool("repair-label-pods"),
			PodLabelKey:   viper.GetString("repair-broken-pod-label-key"),
			PodLabelValue: viper.GetString("repair-broken-pod-label-value"),
		},
	}

	if nodeName := viper.GetString("repair-node-name"); nodeName != "" {
		filters.FieldSelectors = fmt.Sprintf("%s=%s,%s", "spec.nodeName", nodeName, filters.FieldSelectors)
	}

	return
}

// Log human-readable output describing the current filter and option selection
func logCurrentOptions(bpr *brokenPodReconciler, options *controllerOptions) {
	if options.RunAsDaemon {
		log.Infof("Controller Option: Running as a Daemon.")
	}
	if bpr.Options.DeletePods {
		log.Info("Controller Option: Deleting broken pods. Pod Labeling deactivated.")
	}
	if bpr.Options.LabelPods && !bpr.Options.DeletePods {
		log.Infof(
			"Controller Option: Labeling broken pods with label %s=%s",
			bpr.Options.PodLabelKey,
			bpr.Options.PodLabelValue,
		)
	}
	if bpr.Filters.SidecarAnnotation != "" {
		log.Infof("Filter option: Only managing pods with an annotation with key %s", bpr.Filters.SidecarAnnotation)
	}
	if bpr.Filters.FieldSelectors != "" {
		log.Infof("Filter option: Only managing pods with field selector %s", bpr.Filters.FieldSelectors)
	}
	if bpr.Filters.LabelSelectors != "" {
		log.Infof("Filter option: Only managing pods with label selector %s", bpr.Filters.LabelSelectors)
	}
	if bpr.Filters.InitContainerName != "" {
		log.Infof("Filter option: Only managing pods where init container is named %s", bpr.Filters.InitContainerName)
	}
	if bpr.Filters.InitContainerTerminationMessage != "" {
		log.Infof("Filter option: Only managing pods where init container termination message is %s", bpr.Filters.InitContainerTerminationMessage)
	}
	if bpr.Filters.InitContainerExitCode != 0 {
		log.Infof("Filter option: Only managing pods where init container exit status is %d", bpr.Filters.InitContainerExitCode)
	}
}
