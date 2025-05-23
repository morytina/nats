package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSampleConfigFile(t *testing.T) {
	config, err := LoadConfig("../../configs/config.yaml")
	t.Logf("%+v", config)
	assert.NoError(t, err)
}
