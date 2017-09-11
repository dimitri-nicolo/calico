/*
Copyright 2014 The Kubernetes Authors.

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

package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/kubectl/cmd/templates"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/util/i18n"
)

var (
	stopLong = templates.LongDesc(i18n.T(`
		Deprecated: Gracefully shut down a resource by name or filename.

		The stop command is deprecated, all its functionalities are covered by delete command.
		See 'kubectl delete --help' for more details.

		Attempts to shut down and delete a resource that supports graceful termination.
		If the resource is scalable it will be scaled to 0 before deletion.`))

	stopExample = templates.Examples(i18n.T(`
		# Shut down foo.
		kubectl stop replicationcontroller foo

		# Stop pods and services with label name=myLabel.
		kubectl stop pods,services -l name=myLabel

		# Shut down the service defined in service.json
		kubectl stop -f service.json

		# Shut down all resources in the path/to/resources directory
		kubectl stop -f path/to/resources`))
)

func NewCmdStop(f cmdutil.Factory, out io.Writer) *cobra.Command {
	options := &resource.FilenameOptions{}

	cmd := &cobra.Command{
		Use:        "stop (-f FILENAME | TYPE (NAME | -l label | --all))",
		Short:      i18n.T("Deprecated: Gracefully shut down a resource by name or filename"),
		Long:       stopLong,
		Example:    stopExample,
		Deprecated: fmt.Sprintf("use %q instead.", "delete"),
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(cmdutil.ValidateOutputArgs(cmd))
			cmdutil.CheckErr(RunStop(f, cmd, args, out, options))
		},
	}
	usage := "of resource(s) to be stopped."
	cmdutil.AddFilenameOptionFlags(cmd, options, usage)
	cmd.Flags().StringP("selector", "l", "", "Selector (label query) to filter on.")
	cmd.Flags().Bool("all", false, "select all resources in the namespace of the specified resource types.")
	cmd.Flags().Bool("ignore-not-found", false, "Treat \"resource not found\" as a successful stop.")
	cmd.Flags().Int("grace-period", -1, "Period of time in seconds given to the resource to terminate gracefully. Ignored if negative.")
	cmd.Flags().Duration("timeout", 0, "The length of time to wait before giving up on a delete, zero means determine a timeout from the size of the object")
	cmdutil.AddOutputFlagsForMutation(cmd)
	cmdutil.AddInclude3rdPartyFlags(cmd)
	return cmd
}

func RunStop(f cmdutil.Factory, cmd *cobra.Command, args []string, out io.Writer, options *resource.FilenameOptions) error {
	cmdNamespace, enforceNamespace, err := f.DefaultNamespace()
	if err != nil {
		return err
	}

	mapper, _ := f.Object()
	r := f.NewBuilder(true).
		ContinueOnError().
		NamespaceParam(cmdNamespace).DefaultNamespace().
		ResourceTypeOrNameArgs(false, args...).
		FilenameParam(enforceNamespace, options).
		SelectorParam(cmdutil.GetFlagString(cmd, "selector")).
		SelectAllParam(cmdutil.GetFlagBool(cmd, "all")).
		Flatten().
		Do()
	if r.Err() != nil {
		return r.Err()
	}
	shortOutput := cmdutil.GetFlagString(cmd, "output") == "name"
	gracePeriod := cmdutil.GetFlagInt(cmd, "grace-period")
	waitForDeletion := false
	if gracePeriod == 0 {
		// To preserve backwards compatibility, but prevent accidental data loss, we convert --grace-period=0
		// into --grace-period=1 and wait until the object is successfully deleted.
		gracePeriod = 1
		waitForDeletion = true
	}
	return ReapResult(r, f, out, false, cmdutil.GetFlagBool(cmd, "ignore-not-found"), cmdutil.GetFlagDuration(cmd, "timeout"), gracePeriod, waitForDeletion, shortOutput, mapper, false)
}
