package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	yaml "github.com/projectcalico/go-yaml-wrapper"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/compliance"
)

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "View a sample render.",
	Long:  "View a sample rendering of Global Report Type output templates",
	Run: func(cmd *cobra.Command, args []string) {
		runViewCmd(args)
	},
	Args: cobra.MinimumNArgs(2),
}

func runViewCmd(args []string) {
	// Extract report type and template name.
	reportType, templateName := args[0], args[1]

	// Always start with local "default" directory, unless specified.
	dirs := defaultDirs
	if len(args) > 2 {
		dirs = args[2:]
	}

	// Get list of yaml files inside the 1st level of given directories.
	for _, dir := range dirs {
		if err := traverseDir(path.Join(dir, manifestsDir), true, ".yaml", func(f string) error {
			clog := log.WithField("file", f)
			if strings.Split(path.Base(f), ".yaml")[0] != reportType {
				clog.Debug("No match, passing")
				return nil
			}
			clog.Debug("Found file, opening")

			contents, err := ioutil.ReadFile(f)
			if err != nil {
				return err
			}

			reportType := api.GlobalReportType{}
			if err := yaml.UnmarshalStrict(contents, &reportType); err != nil {
				return err
			}

			for _, tmpl := range append(reportType.Spec.DownloadTemplates, reportType.Spec.UISummaryTemplate) {
				clog2 := clog.WithField("tmpl", tmpl.Name)
				if strings.Split(tmpl.Name, ".")[0] != templateName {
					clog2.Debug("No match, passing")
					continue
				}

				clog2.Debug("Template found, rendering")
				rendered, err := compliance.RenderTemplate(tmpl.Template, &compliance.ReportDataSample)
				if err != nil {
					return err
				}

				fmt.Println(rendered)
				os.Exit(0)
			}

			log.Fatal("Requested template does not exist")
			return nil
		}); err != nil {
			log.WithError(err).Fatal("Fatal error occurred while attempting to view rendered manifest")
		}
	}

	log.Fatal("Requested report type does not exist")
}
