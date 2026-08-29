package openai

import "testing"

func Test_parseStage1JSON_arrayFields(t *testing.T) {
	got, err := parseStage1JSON(`{"category":"gpu","fields":[{"key":"model","value":"RTX 3080"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Category != "gpu" || len(got.Fields) != 1 || got.Fields[0].Key != "model" || got.Fields[0].Value != "RTX 3080" {
		t.Fatalf("%+v", got)
	}
}

func Test_parseStage1JSON_objectFields(t *testing.T) {
	// Gemini json_object が fields を {key:value} で返すことがある。
	got, err := parseStage1JSON(`{"category":"server","condition":"中古","shipping_free":false,"fields":{"server_model":"Dell R740","cpu_model_line":"Xeon"},"missing_keys":[]}`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Category != "server" {
		t.Fatalf("%+v", got)
	}
	want := map[string]string{"server_model": "Dell R740", "cpu_model_line": "Xeon"}
	if len(got.Fields) != len(want) {
		t.Fatalf("fields=%+v", got.Fields)
	}
	for _, f := range got.Fields {
		if want[f.Key] != f.Value {
			t.Fatalf("fields=%+v", got.Fields)
		}
	}
}

func Test_parseStage1JSON_flexibleScalars(t *testing.T) {
	got, err := parseStage1JSON(`{"category":{"label":"server"},"condition":{"value":"中古"},"shipping_free":"false","missing_keys":[{"value":"cpu_model_line"}],"fields":[{"key":{"value":"server_model"},"value":{"label":"R740"}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Category) != "server" || string(got.Condition) != "中古" {
		t.Fatalf("%+v", got)
	}
	if got.FreeShipping.ptr() == nil || *got.FreeShipping.ptr() {
		t.Fatalf("shipping_free=%v", got.FreeShipping.ptr())
	}
	if len(got.MissingKeys) != 1 || got.MissingKeys[0] != "cpu_model_line" {
		t.Fatalf("missing_keys=%v", got.MissingKeys)
	}
	if len(got.Fields) != 1 || got.Fields[0].Key != "server_model" || got.Fields[0].Value != "R740" {
		t.Fatalf("fields=%+v", got.Fields)
	}
}

func Test_parseStage2JSON_flexibleFields(t *testing.T) {
	got, err := parseStage2JSON(`{"image_fields":[{"key":{"value":"model"},"value":{"label":"RTX 3080"},"confidence":{"value":"high"}}],"visible_model_numbers":[{"label":"RTX3080"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ImageFields) != 1 {
		t.Fatalf("%+v", got.ImageFields)
	}
	f := got.ImageFields[0]
	if string(f.Key) != "model" || string(f.Value) != "RTX 3080" || string(f.Confidence) != "high" {
		t.Fatalf("%+v", f)
	}
	if len(got.VisibleModelNumbers) != 1 || got.VisibleModelNumbers[0] != "RTX3080" {
		t.Fatalf("%+v", got.VisibleModelNumbers)
	}
}

func Test_parseAgentFieldsJSON_objectDone(t *testing.T) {
	got, err := parseAgentFieldsJSON(`{"fields":{"server_model":"R740"},"done":{"value":true}}`)
	if err != nil {
		t.Fatal(err)
	}
	if !bool(got.Done) || len(got.Fields) != 1 || got.Fields[0].Key != "server_model" {
		t.Fatalf("%+v", got)
	}
}

func Test_parseAgentFieldsJSON_objectFields(t *testing.T) {
	got, err := parseAgentFieldsJSON(`{"fields":{"server_model":"R740"},"done":true}`)
	if err != nil {
		t.Fatal(err)
	}
	if !bool(got.Done) || len(got.Fields) != 1 || got.Fields[0].Key != "server_model" || got.Fields[0].Value != "R740" {
		t.Fatalf("%+v", got)
	}
}

func Test_parseStage1JSON_nullAndEmptyObjectFields(t *testing.T) {
	got, err := parseStage1JSON(`{"category":"other","fields":null}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Fields) != 0 {
		t.Fatalf("%+v", got)
	}
	got, err = parseStage1JSON(`{"category":"other","fields":{}}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Fields) != 0 {
		t.Fatalf("%+v", got)
	}
}
