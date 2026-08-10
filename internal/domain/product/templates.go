package product

import "log"

// FieldTemplate は Category ごとのスペック欄項目定義。
type FieldTemplate struct {
	Key    string
	Label  string
	Inline bool
}

// TemplatesFor は Category に対応する FieldTemplate 一覧を返す。
func TemplatesFor(cat Category) []FieldTemplate {
	switch cat {
	case CategoryServer:
		return serverTemplate
	case CategoryRackRail:
		return rackRailTemplate
	case CategoryNIC:
		return nicTemplate
	case CategoryStorage:
		return storageTemplate
	case CategoryGPU:
		return gpuTemplate
	case CategoryCPU:
		return cpuTemplate
	case CategoryMemory:
		return memoryTemplate
	case CategoryServerRack:
		return serverRackTemplate
	case CategoryUPS:
		return upsTemplate
	case CategoryNetwork:
		return networkTemplate
	case CategoryDesktopNUC:
		return desktopNUCTemplate
	case CategoryOther:
		return otherTemplate
	default:
		return otherTemplate
	}
}

// TemplateKeys は Category の FieldTemplate キー一覧を返す。
func TemplateKeys(cat Category) []string {
	defs := TemplatesFor(cat)
	keys := make([]string, len(defs))
	for i, d := range defs {
		keys[i] = d.Key
	}
	return keys
}

// CanonicalFieldKey はカテゴリのテンプレートキーに正規化する。未知キーは空文字。
func CanonicalFieldKey(cat Category, key string) string {
	if key == "" {
		return ""
	}
	for _, d := range TemplatesFor(cat) {
		if d.Key == key {
			return key
		}
	}
	log.Printf("[product] unknown field key %q for category %s", key, cat)
	return ""
}

// ValidateFields はテンプレートに定義されたキーのみをテンプレート順で返す。
func ValidateFields(cat Category, fields []Field) []Field {
	defs := TemplatesFor(cat)

	byKey := make(map[string]string)
	for _, f := range fields {
		if f.Key == "" {
			continue
		}
		canon := CanonicalFieldKey(cat, f.Key)
		if canon == "" {
			continue
		}
		if _, ok := byKey[canon]; ok {
			continue
		}
		byKey[canon] = f.Value
	}

	out := make([]Field, 0, len(defs))
	for _, d := range defs {
		if v, ok := byKey[d.Key]; ok {
			out = append(out, Field{Key: d.Key, Value: v})
		}
	}
	return out
}

// FieldValueMap は Fields をキー→値のマップに変換する。
func FieldValueMap(fields []Field) map[string]string {
	m := make(map[string]string, len(fields))
	for _, f := range fields {
		if f.Key != "" {
			m[f.Key] = f.Value
		}
	}
	return m
}

var serverTemplate = []FieldTemplate{
	{Key: "server_model", Label: "サーバー機種名", Inline: false},
	{Key: "cpu_model_line", Label: "CPU型番", Inline: false},
	{Key: "core_thread_info", Label: "CPUコア数/スレッド数", Inline: false},
	{Key: "socket_count", Label: "ソケット数", Inline: true},
	{Key: "memory_info", Label: "メモリー", Inline: false},
	{Key: "storage_info", Label: "ストレージ", Inline: false},
	{Key: "gpu", Label: "GPU", Inline: false},
	{Key: "other_notes", Label: "その他特記事項", Inline: false},
}

var rackRailTemplate = []FieldTemplate{
	{Key: "compatible_models", Label: "対応機種/メーカー", Inline: false},
	{Key: "rail_type", Label: "レール種別", Inline: true},
	{Key: "depth_u", Label: "深さ/U", Inline: true},
	{Key: "quantity", Label: "個数", Inline: true},
	{Key: "other_notes", Label: "その他特記事項", Inline: false},
}

var nicTemplate = []FieldTemplate{
	{Key: "model", Label: "型番", Inline: false},
	{Key: "interface_speed", Label: "IF/速度", Inline: true},
	{Key: "port_count", Label: "ポート数", Inline: true},
	{Key: "bus", Label: "バス", Inline: true},
	{Key: "other_notes", Label: "その他特記事項", Inline: false},
}

var storageTemplate = []FieldTemplate{
	{Key: "model", Label: "型番", Inline: false},
	{Key: "media_type", Label: "メディア種別", Inline: true},
	{Key: "interface", Label: "接続IF", Inline: true},
	{Key: "capacity", Label: "容量", Inline: true},
	{Key: "quantity", Label: "個数", Inline: true},
	{Key: "other_notes", Label: "その他特記事項", Inline: false},
}

var gpuTemplate = []FieldTemplate{
	{Key: "model", Label: "型番", Inline: false},
	{Key: "vram", Label: "VRAM", Inline: true},
	{Key: "quantity", Label: "個数", Inline: true},
	{Key: "other_notes", Label: "その他特記事項", Inline: false},
}

var cpuTemplate = []FieldTemplate{
	{Key: "model", Label: "型番", Inline: false},
	{Key: "core_thread_info", Label: "コア/スレッド", Inline: true},
	{Key: "frequency", Label: "周波数", Inline: true},
	{Key: "quantity", Label: "個数", Inline: true},
	{Key: "socket_gen", Label: "ソケット/世代", Inline: true},
	{Key: "other_notes", Label: "その他特記事項", Inline: false},
}

var memoryTemplate = []FieldTemplate{
	{Key: "model_spec", Label: "型番/規格", Inline: false},
	{Key: "capacity", Label: "容量", Inline: true},
	{Key: "quantity", Label: "枚数", Inline: true},
	{Key: "ecc_type", Label: "ECC等", Inline: true},
	{Key: "other_notes", Label: "その他特記事項", Inline: false},
}

var serverRackTemplate = []FieldTemplate{
	{Key: "model", Label: "型番/メーカー", Inline: false},
	{Key: "u_height", Label: "U数", Inline: true},
	{Key: "depth", Label: "深さ", Inline: true},
	{Key: "accessories", Label: "付属品", Inline: false},
	{Key: "other_notes", Label: "その他特記事項", Inline: false},
}

var upsTemplate = []FieldTemplate{
	{Key: "model", Label: "型番", Inline: false},
	{Key: "capacity", Label: "容量(VA/W)", Inline: true},
	{Key: "battery_status", Label: "バッテリー状態", Inline: true},
	{Key: "other_notes", Label: "その他特記事項", Inline: false},
}

var networkTemplate = []FieldTemplate{
	{Key: "device_type", Label: "機器種別(SW/RT/AP)", Inline: true},
	{Key: "model", Label: "型番", Inline: false},
	{Key: "port_spec", Label: "ポート仕様", Inline: true},
	{Key: "firmware_license", Label: "FW/ライセンス", Inline: false},
	{Key: "other_notes", Label: "その他特記事項", Inline: false},
}

var desktopNUCTemplate = []FieldTemplate{
	{Key: "model", Label: "モデル名", Inline: false},
	{Key: "cpu", Label: "CPU", Inline: false},
	{Key: "memory", Label: "メモリ", Inline: true},
	{Key: "storage", Label: "ストレージ", Inline: true},
	{Key: "gpu", Label: "GPU", Inline: true},
	{Key: "other_notes", Label: "その他特記事項", Inline: false},
}

var otherTemplate = []FieldTemplate{
	{Key: "summary", Label: "概要", Inline: false},
	{Key: "other_notes", Label: "その他", Inline: false},
}

// CategoryFieldKeysForPrompt はプロンプト用に Category 別 FieldTemplate キー一覧を返す。
func CategoryFieldKeysForPrompt() map[Category][]string {
	out := make(map[Category][]string, len(AllCategories))
	for _, cat := range AllCategories {
		out[cat] = TemplateKeys(cat)
	}
	return out
}
