package pools

import (
	"testing"

	"github.com/hostodo/odo-cli/v2/pkg/api"
)

func TestFindPoolTier_ByIDAndName(t *testing.T) {
	tiers := []api.PoolTier{
		{ID: 12, Name: "Hostodo Nano"},
		{ID: 13, Name: "Hostodo Micro"},
	}

	got, err := findPoolTier(tiers, "12")
	if err != nil || got == nil || got.Name != "Hostodo Nano" {
		t.Fatalf("by id: got %+v err %v", got, err)
	}

	got, err = findPoolTier(tiers, "micro")
	if err != nil || got == nil || got.ID != 13 {
		t.Fatalf("by substring: got %+v err %v", got, err)
	}

	_, err = findPoolTier(tiers, "Hostodo")
	if err == nil {
		t.Fatal("expected ambiguous error")
	}
}
