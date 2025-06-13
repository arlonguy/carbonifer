package cmd

import (
	"bufio"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/carboniferio/carbonifer/internal/data" // <-- add this import
	"github.com/carboniferio/carbonifer/internal/estimate"
	"github.com/carboniferio/carbonifer/internal/output"
	"github.com/carboniferio/carbonifer/internal/plan"
	"github.com/carboniferio/carbonifer/internal/terraform"
	"github.com/shopspring/decimal" // <-- add this import
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var testPlanCmdHasRun = false

// planCmd represents the plan command
var planCmd = &cobra.Command{
	Use: "plan",
	Long: `Estimate CO2 from your infrastructure code.

The 'plan' command optionally takes a single argument:

    directory : 
		- default: current directory
		- directory: a terraform project directory
		- file: a terraform plan file (raw or json)
Example usages:
	carbonifer plan
	carbonifer plan /path/to/terraform/project
	carbonifer plan /path/to/terraform/plan.json
	carbonifer plan /path/to/terraform/plan.tfplan`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		testPlanCmdHasRun = true
		log.Debug("Running command 'plan'")

		workdir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}

		input := workdir
		if len(args) != 0 {
			input = args[0]
			if !filepath.IsAbs(input) {
				input = filepath.Join(workdir, input)
			}
		}

		// Generate or Read Terraform plan
		tfPlan, err := terraform.CarboniferPlan(input)
		if err != nil {
			log.Fatal(err)
		}

		// Read resources from terraform plan
		resources, err := plan.GetResources(tfPlan)
		if err != nil {
			errW := errors.Wrap(err, "Failed to get resources from terraform plan")
			log.Panic(errW)
		}

		// New code for forecast file
		forecastFile := viper.GetString("carbon_intensity_file")
		var forecastCarbonIntensity *decimal.Decimal
		var forecastRegion string

		if forecastFile != "" {
			value, region, err := data.ReadForecastCarbonIntensity(forecastFile)
			if err != nil {
				log.Warnf("Error loading forecast carbon intensity, falling back to default: %v", err)
			} else {
				d := decimal.NewFromFloat(value)
				forecastCarbonIntensity = &d
				forecastRegion = region
				log.Infof("Using forecast carbon intensity from %s (region: %s): %.6f gCO2eq/Wh", forecastFile, forecastRegion, value)
			}
		} else {
			log.Info("No forecast carbon intensity file provided — using static carbon intensities only")
		}

		// Estimate CO2 emissions with forecast params
		estimations := estimate.EstimateResources(resources, forecastCarbonIntensity, forecastRegion)

		// Generate report
		// Generate report
		reportText := ""
		if viper.Get("out.format") == "json" {
			reportText = output.GenerateReportJSON(estimations)
		} else {
			reportText = output.GenerateReportText(estimations, forecastCarbonIntensity != nil)
		}

		// Print out report
		outFile := viper.Get("out.file").(string)
		if outFile == "" {
			log.Debug("output : stdout")
			cmd.SetOut(os.Stdout)
			cmd.Println(reportText)
		} else {
			log.Debug("output :", outFile)
			f, err := os.Create(outFile)
			if err != nil {
				log.Fatal(err)
			}
			outWriter := bufio.NewWriter(f)
			_, err = outWriter.WriteString(reportText)
			if err != nil {
				log.Fatal(err)
			}
			err = outWriter.Flush()
			if err != nil {
				log.Fatal(err)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(planCmd)

	// Add CLI flag for forecast carbon intensity file
	planCmd.Flags().String("carbon-intensity-file", "", "Path to JSON file with forecast carbon intensity data")
	viper.BindPFlag("carbon_intensity_file", planCmd.Flags().Lookup("carbon-intensity-file"))
}
