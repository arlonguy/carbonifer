package data

import (
	"embed"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

//go:embed data/*
var data embed.FS

// ReadDataFile reads a file from the data directory
func ReadDataFile(filename string) []byte {
	dataPath := viper.GetString("data.path")
	if dataPath != "" {
		// If the environment variable is set, read from the specified file
		filePath := filepath.Join(dataPath, filename)
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			log.Debugf("  reading datafile '%v' from: %v", filename, filePath)
			data, err := os.ReadFile(filePath)
			if err != nil {
				log.Fatal(err)
			}
			return data
		}
		return readEmbeddedFile(filename)

	}
	return readEmbeddedFile(filename)
}

func readEmbeddedFile(filename string) []byte {
	log.Debugf("  reading datafile '%v' embedded", filename)
	data, err := fs.ReadFile(data, "data/"+filename)
	if err != nil {
		errW := errors.Wrap(err, "cannot read embedded data file")
		log.Fatal(errW)
	}
	return data
}

type ForecastFile struct {
	Region string          `json:"region"`
	Data   []ForecastEntry `json:"data"`
}

type ForecastEntry struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

// ReadForecastCarbonIntensity reads a forecast JSON file and returns:
// 1) the average carbon intensity (gCO2eq/Wh)
// 2) the region the forecast applies to
func ReadForecastCarbonIntensity(filename string) (float64, string, error) {
	log.Infof("Reading forecast carbon intensity from: %s", filename)

	// reads the file at filename
	fileData, err := os.ReadFile(filename)
	if err != nil {
		return 0.0, "", errors.Wrap(err, "failed to read forecast carbon intensity file")
	}

	// parses JSON array into []ForecastEntry
	var forecast ForecastFile
	if err := json.Unmarshal(fileData, &forecast); err != nil {
		return 0.0, "", errors.Wrap(err, "failed to parse forecast carbon intensity JSON")
	}

	if len(forecast.Data) == 0 {
		return 0.0, "", errors.New("forecast carbon intensity file is empty")
	}

	// loops over entries, sums value
	// divide by 1000 because WattTime provides gCO2eq/kWh while Carbonifer expects gCO2eq/Wh
	var sum float64
	for _, entry := range forecast.Data {
		// Convert gCO2eq/kWh -> gCO2eq/Wh
		sum += entry.Value / 1000.0
	}

	// logs helpful messages and returns the average as float64
	avg := sum / float64(len(forecast.Data))
	log.Infof("Computed average forecast carbon intensity: %.6f gCO2eq/Wh for region %s", avg, forecast.Region)

	return avg, forecast.Region, nil
}
