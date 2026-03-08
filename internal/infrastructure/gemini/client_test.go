package gemini

import (
	"testing"
)

func Test_extractJSONFromResponse(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "plain json",
			text: `{"cpu_model":"Xeon E3-1230 v6","core_count":4}`,
			want: `{"cpu_model":"Xeon E3-1230 v6","core_count":4}`,
		},
		{
			name: "json in markdown code block",
			text: "```json\n{\"cpu_model\":\"Xeon E3-1230 v6\",\"core_count\":4}\n```",
			want: `{"cpu_model":"Xeon E3-1230 v6","core_count":4}`,
		},
		{
			name: "plain code block without json",
			text: "```\n{\"cpu_model\":\"Xeon\"}\n```",
			want: `{"cpu_model":"Xeon"}`,
		},
		{
			name: "with leading trailing whitespace",
			text: "  \n  ```json\n  {\"memory_gb\":24}\n  ```  \n",
			want: `{"memory_gb":24}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSONFromResponse(tt.text)
			if got != tt.want {
				t.Errorf("extractJSONFromResponse() = %q, want %q", got, tt.want)
			}
		})
	}
}
