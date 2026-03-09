package gemini

import (
	"strings"
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

func TestCleanHTMLToText_RemovesUnnecessaryTagsAndNormalizesWhitespace(t *testing.T) {
	html := `
<!DOCTYPE html>
<html>
  <head>
    <title>テストタイトル</title>
    <style>body { color: red; }</style>
  </head>
  <body>
    <h1>見出し</h1>
    <p>説明 その1</p>
    <script>alert("x")</script>
    <p>説明   その2</p>
    <iframe src="https://example.com"></iframe>
    <noscript>スクリプト無効時の説明</noscript>
  </body>
</html>`

	got, err := CleanHTMLToText(html)
	if err != nil {
		t.Fatalf("CleanHTMLToText() error = %v", err)
	}

	if got == "" {
		t.Fatalf("CleanHTMLToText() returned empty string")
	}

	// script / iframe / noscript 内のテキストは含まれていないこと。
	if strings.Contains(got, "alert") {
		t.Errorf("CleanHTMLToText() should remove script content, got = %q", got)
	}
	if strings.Contains(got, "スクリプト無効時の説明") {
		t.Errorf("CleanHTMLToText() should remove noscript content, got = %q", got)
	}

	// 連続する空白がそのまま残っていないこと（2つ以上のスペースを含まないこと）をざっくり検証。
	if strings.Contains(got, "  ") {
		t.Errorf("CleanHTMLToText() should normalize consecutive spaces, got = %q", got)
	}

	// 前後の空白が削除されていること。
	if strings.HasPrefix(got, " ") || strings.HasSuffix(got, " ") {
		t.Errorf("CleanHTMLToText() should trim leading/trailing spaces, got = %q", got)
	}
}

func TestHTMLToMarkdown_ConvertsStructureAndDropsImages(t *testing.T) {
	html := `
<h1>タイトル</h1>
<p>説明テキストです。</p>
<ul>
  <li>項目1</li>
  <li>項目2</li>
</ul>
<img src="https://example.com/image.jpg" alt="テスト画像" />
`

	got, err := HTMLToMarkdown(html)
	if err != nil {
		t.Fatalf("HTMLToMarkdown() error = %v", err)
	}

	if got == "" {
		t.Fatalf("HTMLToMarkdown() returned empty string")
	}

	// 見出しやリストなどの構造がある程度維持されていることを緩く確認する。
	if !strings.Contains(got, "タイトル") {
		t.Errorf("HTMLToMarkdown() markdown should contain heading text, got = %q", got)
	}
	if !strings.Contains(got, "項目1") || !strings.Contains(got, "項目2") {
		t.Errorf("HTMLToMarkdown() markdown should contain list items, got = %q", got)
	}

	// 画像のURLやaltテキストは出力に含まれないこと。
	if strings.Contains(got, "image.jpg") || strings.Contains(got, "テスト画像") {
		t.Errorf("HTMLToMarkdown() markdown should drop img tags, got = %q", got)
	}
}
