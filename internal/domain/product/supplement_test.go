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

func TestFilterSupplementEligibleKeys(t *testing.T) {
	got := FilterSupplementEligibleKeys(CategoryServer, []string{
		"server_model", "cpu_model_line", "memory_info", "other_notes",
	})
	want := []string{"server_model", "other_notes"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}
