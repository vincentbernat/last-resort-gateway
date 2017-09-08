package reporter

import (
	"testing"

	"gopkg.in/yaml.v2"

	"lrg/helpers"
	"lrg/reporter/logger"
)

func TestEmptyConfiguration(t *testing.T) {
	var configuration Configuration
	err := yaml.Unmarshal([]byte("{}"), &configuration)
	if err != nil {
		t.Fatalf("Unmarshal(%q) error:\n%+v",
			"{}", err)
	}
	expected := Configuration{
		Logging: logger.DefaultConfiguration,
	}
	if diff := helpers.Diff(configuration, expected); diff != "" {
		t.Errorf("Unmarshal(%q) (-got +want):\n%s", "{}", diff)
	}
}
