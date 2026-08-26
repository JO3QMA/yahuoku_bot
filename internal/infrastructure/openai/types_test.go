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
	got, err := parseStage1JSON(`{"category":"server","condition":"中古","shipping_free":false,"fields":{"server_model":"Dell R740","cpu_model_line":"Xeon"},"missing_keys":[],"candidate_queries":[]}`)
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

func Test_parseProductJSON_objectFields(t *testing.T) {
	got, err := parseProductJSON(`{"category":"gpu","fields":{"model":"RTX 3080"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Fields) != 1 || got.Fields[0].Key != "model" || got.Fields[0].Value != "RTX 3080" {
		t.Fatalf("%+v", got)
	}
}

func Test_parseAgentFieldsJSON_objectFields(t *testing.T) {
	got, err := parseAgentFieldsJSON(`{"fields":{"server_model":"R740"},"done":true}`)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Done || len(got.Fields) != 1 || got.Fields[0].Key != "server_model" || got.Fields[0].Value != "R740" {
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
