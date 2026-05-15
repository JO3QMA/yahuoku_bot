package spec

// Spec はLLMが商品説明・タイトルから抽出したPCスペック等の構造化データ。
// 各項目は独立しており、不明時は空文字または0とする。
type Spec struct {
	// CPU型番 (x個数) (周波数)。例: "Xeon E-2224 (x1) (3.4GHz)"
	CPUModelLine string `json:"cpu_model_line"`
	// CPUコア数/スレッド数。例: "4コア/4スレッド"
	CoreThreadInfo string `json:"core_thread_info"`
	// ソケット数。不明時は0
	SocketCount int `json:"socket_count"`
	// メモリー容量/枚数。例: "16GB" または "16GB x2"
	MemoryInfo string `json:"memory_info"`
	// ストレージ種別。例: "SATA HDD", "NVMe SSD"
	StorageType string `json:"storage_type"`
	// ストレージ容量。例: "1TB x2"
	StorageCapacity string `json:"storage_capacity"`
	// その他特記事項
	OtherNotes string `json:"other_notes"`

	Condition    string `json:"condition"`     // "新品" / "中古" / "不明"
	ShippingFree *bool  `json:"shipping_free"` // true=送料無料, false=落札者負担, nil=不明
}
