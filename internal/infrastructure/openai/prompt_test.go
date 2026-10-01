package openai

import (
	"strings"
	"testing"
)

func TestBuildStage1Prompt_cpuFieldFormat(t *testing.T) {
	p := buildStage1Prompt("t", "d")
	if !strings.Contains(p, "(n CPU/m Core/o Thread)") {
		t.Fatal("stage1 should enforce core_thread_info format")
	}
	if !strings.Contains(p, "(SkyLake)") {
		t.Fatal("stage1 should show cpu_model_line generation example")
	}
	if strings.Contains(p, "@4.25GHz x1") {
		t.Fatal("stage1 should not show cpu quantity suffix example")
	}
}
