package product

import "testing"

func TestIsSupplementEligibleKey_rejectsInstalledConfiguration(t *testing.T) {
	if IsSupplementEligibleKey(CategoryServer, "cpu_model_line") {
		t.Fatal("cpu_model_line should not be supplement eligible")
	}
	if !IsSupplementEligibleKey(CategoryServer, "server_model") {
		t.Fatal("server_model should be supplement eligible")
	}
}
