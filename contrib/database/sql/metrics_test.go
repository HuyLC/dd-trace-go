// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package sql

import (
	"sync"
	"testing"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/globalconfig"
)

func (cfg *config) applyTags() {
	cfg.serviceName = "my-svc"
	cfg.tags = make(map[string]interface{})
	cfg.tags["tag"] = "value"
}

func setGlobalCfgTags() {
	globalconfig.SetStatsTags([]string{"globaltag:globalvalue"})
}

func resetGlobalConfig() {
	globalconfig.SetStatsTags([]string{})
}

// Test that statsTags(*config) returns tags from the provided *config + whatever is on the globalconfig
func TestStatsTags(t *testing.T) {
	t.Run("default none", func(t *testing.T) {
		resetGlobalConfig()
		cfg := new(config)
		tags := statsTags(cfg)
		assert.Len(t, tags, 0)
	})
	t.Run("cfg only", func(t *testing.T) {
		resetGlobalConfig()
		cfg := new(config)
		cfg.applyTags()
		tags := statsTags(cfg)
		assert.Len(t, tags, 2)
		assert.Contains(t, tags, "service:my-svc")
		assert.Contains(t, tags, "tag:value")
	})
	t.Run("inherit globalconfig", func(t *testing.T) {
		resetGlobalConfig()
		cfg := new(config)
		setGlobalCfgTags()
		tags := statsTags(cfg)
		assert.Len(t, tags, 1)
		assert.Contains(t, tags, "globaltag:globalvalue")
	})
	t.Run("both", func(t *testing.T) {
		resetGlobalConfig()
		cfg := new(config)
		cfg.applyTags()
		setGlobalCfgTags()
		tags := statsTags(cfg)
		assert.Len(t, tags, 3)
		assert.Contains(t, tags, "globaltag:globalvalue")
		assert.Contains(t, tags, "service:my-svc")
		assert.Contains(t, tags, "tag:value")
	})
	resetGlobalConfig()
}

func TestPollDBStatsStop(t *testing.T) {
	driverName := "postgres"
	Register(driverName, &pq.Driver{}, WithServiceName("postgres-test"), WithAnalyticsRate(0.2))
	defer unregister(driverName)
	db, err := Open(driverName, "postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		pollDBStats(&statsd.NoOpClientDirect{}, db, stop)
	}()
	close(stop)
	wg.Wait()
}
