// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package tracer

import (
	"fmt"
	"testing"

	"gopkg.in/DataDog/dd-trace-go.v1/internal/globalconfig"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/telemetry"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/telemetry/telemetrytest"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"

	"github.com/stretchr/testify/assert"
)

func TestTelemetryEnabled(t *testing.T) {
	t.Run("tracer start", func(t *testing.T) {
		telemetryClient := new(telemetrytest.RecordClient)
		defer telemetry.MockClient(telemetryClient)()

		Start(
			WithDebugMode(true),
			WithService("test-serv"),
			WithEnv("test-env"),
			WithRuntimeMetrics(),
			WithPeerServiceMapping("key", "val"),
			WithPeerServiceDefaults(true),
			WithDebugStack(false),
			WithHeaderTags([]string{"key:val", "key2:val2"}),
			WithSamplingRules(
				[]SamplingRule{TagsResourceRule(
					map[string]string{"tag-a": "tv-a??"},
					"resource-*", "op-name", "test-serv", 0.1),
				},
			),
		)
		defer globalconfig.SetServiceName("")
		defer Stop()

		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "trace_debug_enabled", true)
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "service", "test-serv")
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "env", "test-env")
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "runtime_metrics_enabled", true)
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "stats_computation_enabled", false)
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "trace_enabled", true)
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "trace_span_attribute_schema", 0)
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "trace_peer_service_defaults_enabled", true)
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "trace_peer_service_mapping", "key:val")
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "debug_stack_enabled", false)
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "orchestrion_enabled", false)
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "trace_sample_rate", nil) // default value is NaN which is sanitized to nil
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "trace_header_tags", "key:val,key2:val2")
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "trace_sample_rules",
			`[{"service":"test-serv","name":"op-name","resource":"resource-*","sample_rate":0.1,"tags":{"tag-a":"tv-a??"}}]`)
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "span_sample_rules", "[]")

		assert.NotZero(t, telemetryClient.Distribution(telemetry.NamespaceGeneral, "init_time", nil).Get())
	})

	t.Run("telemetry customer or dynamic rules", func(t *testing.T) {
		rule := TagsResourceRule(map[string]string{"tag-a": "tv-a??"}, "resource-*", "op-name", "test-serv", 0.1)

		for _, prov := range provenances {
			if prov == Local {
				continue
			}
			rule.Provenance = prov

			telemetryClient := new(telemetrytest.RecordClient)
			defer telemetry.MockClient(telemetryClient)()
			Start(WithService("test-serv"),
				WithSamplingRules([]SamplingRule{rule}),
			)
			defer globalconfig.SetServiceName("")
			defer Stop()

			telemetrytest.CheckConfig(t, telemetryClient.Configuration, "trace_sample_rules",
				fmt.Sprintf(`[{"service":"test-serv","name":"op-name","resource":"resource-*","sample_rate":0.1,"tags":{"tag-a":"tv-a??"},"provenance":"%s"}]`, prov.String()))
		}
	})

	t.Run("telemetry local rules", func(t *testing.T) {
		rules := []SamplingRule{
			TagsResourceRule(map[string]string{"tag-a": "tv-a??"}, "resource-*", "op-name", "test-serv", 0.1),
			// Span rules can have only local provenance for now.
			SpanNameServiceRule("op-name", "test-serv", 0.1),
		}

		for i := range rules {
			rules[i].Provenance = Local
		}

		telemetryClient := new(telemetrytest.RecordClient)
		defer telemetry.MockClient(telemetryClient)()
		Start(WithService("test-serv"),
			WithSamplingRules(rules),
		)
		defer globalconfig.SetServiceName("")
		defer Stop()

		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "trace_sample_rules",
			`[{"service":"test-serv","name":"op-name","resource":"resource-*","sample_rate":0.1,"tags":{"tag-a":"tv-a??"}}]`)
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "span_sample_rules",
			`[{"service":"test-serv","name":"op-name","sample_rate":0.1}]`)
	})

	t.Run("tracer start with empty rules", func(t *testing.T) {
		telemetryClient := new(telemetrytest.RecordClient)
		defer telemetry.MockClient(telemetryClient)()

		t.Setenv("DD_TRACE_SAMPLING_RULES", "")
		t.Setenv("DD_SPAN_SAMPLING_RULES", "")
		Start()
		defer globalconfig.SetServiceName("")
		defer Stop()

		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "trace_sample_rules", "[]")
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "span_sample_rules", "[]")
	})

	t.Run("profiler start, tracer start", func(t *testing.T) {
		telemetryClient := new(telemetrytest.RecordClient)
		defer telemetry.MockClient(telemetryClient)()
		profiler.Start()
		defer profiler.Stop()
		Start(
			WithService("test-serv"),
		)
		defer globalconfig.SetServiceName("")
		defer Stop()
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "service", "test-serv")
	})
	t.Run("orchestrion telemetry", func(t *testing.T) {
		telemetryClient := new(telemetrytest.RecordClient)
		defer telemetry.MockClient(telemetryClient)()

		Start(WithOrchestrion(map[string]string{"k1": "v1", "k2": "v2"}))
		defer Stop()

		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "orchestrion_enabled", true)
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "orchestrion_k1", "v1")
		telemetrytest.CheckConfig(t, telemetryClient.Configuration, "orchestrion_k2", "v2")
	})
}
