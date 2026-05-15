package spec

import (
	"encoding/json"
	"testing"
)

func TestSpec_JSONRoundTrip(t *testing.T) {
	sf := true
	s := &Spec{
		CPUModelLine: "c", CoreThreadInfo: "ct", SocketCount: 2,
		MemoryInfo: "m", StorageType: "st", StorageCapacity: "sc",
		OtherNotes: "o", Condition: "新品", ShippingFree: &sf,
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var out Spec
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.CPUModelLine != "c" || out.SocketCount != 2 {
		t.Fatal(out)
	}
}
