package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSampleConfigFile(t *testing.T) {
	config, err := LoadConfig("../../configs/config.yaml")
	assert.NoError(t, err)
	if assert.NotNil(t, config) {
		assert.Equal(t, "kr-west1", config.Region)
		assert.Equal(t, "dev2", config.Env)
		assert.Equal(t, 5, config.Nats.ConnPoolCnt)
		assert.Equal(t, "localhost:6379", config.Valkey.Addr)
	}
}
