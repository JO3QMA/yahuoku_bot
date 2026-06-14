package product

// Category は Product の種別判別軸。
type Category string

const (
	CategoryServer     Category = "server"
	CategoryRackRail   Category = "rack_rail"
	CategoryNIC        Category = "nic"
	CategoryStorage    Category = "storage"
	CategoryGPU        Category = "gpu"
	CategoryCPU        Category = "cpu"
	CategoryMemory     Category = "memory"
	CategoryServerRack Category = "server_rack"
	CategoryUPS        Category = "ups"
	CategoryNetwork    Category = "network"
	CategoryDesktopNUC Category = "desktop_nuc"
	CategoryOther      Category = "other"
)

// AllCategories は判別対象の全 Category。
var AllCategories = []Category{
	CategoryServer,
	CategoryRackRail,
	CategoryNIC,
	CategoryStorage,
	CategoryGPU,
	CategoryCPU,
	CategoryMemory,
	CategoryServerRack,
	CategoryUPS,
	CategoryNetwork,
	CategoryDesktopNUC,
	CategoryOther,
}

// DisplayName はEmbed表示用の日本語名を返す。
func (c Category) DisplayName() string {
	switch c {
	case CategoryServer:
		return "サーバー"
	case CategoryRackRail:
		return "ラックマウントレール"
	case CategoryNIC:
		return "NIC"
	case CategoryStorage:
		return "SSD/HDD"
	case CategoryGPU:
		return "GPU"
	case CategoryCPU:
		return "CPU"
	case CategoryMemory:
		return "Mem"
	case CategoryServerRack:
		return "サーバーラック"
	case CategoryUPS:
		return "UPS"
	case CategoryNetwork:
		return "NW SW/RT/AP"
	case CategoryDesktopNUC:
		return "デスクトップPC/NUC"
	case CategoryOther:
		return "その他"
	default:
		return "その他"
	}
}

// ParseCategory は文字列をCategoryに変換する。未知値は CategoryOther。
func ParseCategory(s string) Category {
	for _, c := range AllCategories {
		if string(c) == s {
			return c
		}
	}
	return CategoryOther
}
