package product

// IsSupplementEligibleKey は ListingEvidence に無い場合でも Web 検索 Supplement で
// 値を入れてよい FieldTemplate キーかどうかを返す（識別・ModelInvariant のみ）。
func IsSupplementEligibleKey(cat Category, key string) bool {
	canon := CanonicalFieldKey(cat, key)
	if canon == "" {
		return false
	}
	for _, k := range SupplementEligibleKeys(cat) {
		if k == canon {
			return true
		}
	}
	return false
}

// SupplementEligibleKeys は Category ごとに Supplement で補完可能なテンプレートキー一覧を返す。
func SupplementEligibleKeys(cat Category) []string {
	switch cat {
	case CategoryServer:
		return []string{"server_model", "storage_info", "other_notes"}
	case CategoryRackRail:
		return []string{"compatible_models", "other_notes"}
	case CategoryMemory:
		return []string{"model_spec", "other_notes"}
	case CategoryOther:
		return []string{"summary", "other_notes"}
	default:
		return []string{"model", "other_notes"}
	}
}

// FilterSupplementEligibleKeys は keys のうち Supplement 対象のみを返す。
func FilterSupplementEligibleKeys(cat Category, keys []string) []string {
	if len(keys) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, k := range keys {
		canon := CanonicalFieldKey(cat, k)
		if canon == "" || !IsSupplementEligibleKey(cat, canon) {
			continue
		}
		if _, ok := seen[canon]; ok {
			continue
		}
		seen[canon] = struct{}{}
		out = append(out, canon)
	}
	return out
}
