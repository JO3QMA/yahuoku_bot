package market

import (
	"strings"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

// IdentityFieldKey は Category に対応する IdentityField キーを返す。
func IdentityFieldKey(cat product.Category) string {
	switch cat {
	case product.CategoryServer:
		return "server_model"
	case product.CategoryRackRail:
		return "compatible_models"
	case product.CategoryNIC, product.CategoryStorage, product.CategoryGPU,
		product.CategoryCPU, product.CategoryNetwork, product.CategoryServerRack,
		product.CategoryUPS, product.CategoryDesktopNUC:
		return "model"
	case product.CategoryMemory:
		return "model_spec"
	case product.CategoryOther:
		return "summary"
	default:
		// 未知カテゴリは summary を試すが、該当フィールドがなければ IdentityValue は ok=false
		return "summary"
	}
}

// IdentityValue は Product から IdentityField の key と value を返す。value が空なら ok=false。
func IdentityValue(p *product.Product) (key, value string, ok bool) {
	if p == nil {
		return "", "", false
	}
	key = IdentityFieldKey(p.Category)
	values := product.FieldValueMap(p.Fields)
	value = strings.TrimSpace(values[key])
	if value == "" || value == "不明" {
		return key, "", false
	}
	return key, value, true
}
