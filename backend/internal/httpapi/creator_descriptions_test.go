package httpapi

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"
)

func TestIntegrationCreatorRequiredDescriptions(t *testing.T) {
	h := setup(t)
	newTestSession(t, h, "descriptions")

	expectValidation := func(method, path, body string) {
		t.Helper()
		data := h.expect(method, path, body, 422)
		var response struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &response); err != nil {
			t.Fatal(err)
		}
		if response.Error.Code != "validation" {
			t.Fatalf("%s %s returned error code %q, want validation", method, path, response.Error.Code)
		}
	}
	readProfile := func() *Profile {
		t.Helper()
		var response struct {
			Profile *Profile `json:"profile"`
		}
		data := h.expect("GET", "/me/profile", "", 200)
		if err := json.Unmarshal([]byte(data), &response); err != nil {
			t.Fatal(err)
		}
		return response.Profile
	}
	readTiers := func() []Tier {
		t.Helper()
		var response struct {
			Tiers []Tier `json:"tiers"`
		}
		data := h.expect("GET", "/me/tiers", "", 200)
		if err := json.Unmarshal([]byte(data), &response); err != nil {
			t.Fatal(err)
		}
		return response.Tiers
	}

	expectValidation("PUT", "/me/profile", `{"nickname":"descriptions","displayName":"First attempt"}`)
	var firstProfile struct {
		Profile jsontext.Value `json:"profile"`
	}
	data := h.expect("GET", "/me/profile", "", 200)
	if err := json.Unmarshal([]byte(data), &firstProfile); err != nil {
		t.Fatal(err)
	}
	if firstProfile.Profile.Kind() != 'n' {
		t.Fatal("expected an explicit null profile after the rejected first save")
	}
	h.expect("PUT", "/me/profile", `{"nickname":"descriptions","displayName":"Original author","description":"  Original biography  "}`, 200)
	tier := testTier(t, h.expect("POST", "/me/tiers", `{"name":"Original tier","description":"  Original benefits  ","priceKopecks":9900}`, 201))
	tierPath := "/me/tiers/" + tier.ID

	for _, description := range []string{"", `,"description":null`, `,"description":42`} {
		profileBody := `{"nickname":"changed","displayName":"Changed author"` + description + "}"
		tierBody := `{"name":"Changed tier","priceKopecks":12900` + description + "}"
		expectValidation("PUT", "/me/profile", profileBody)
		expectValidation("POST", "/me/tiers", tierBody)
		expectValidation("PUT", tierPath, tierBody)
	}
	if profile := readProfile(); profile == nil || profile.Nickname != "descriptions" || profile.DisplayName != "Original author" || profile.Description != "Original biography" {
		t.Fatalf("unexpected stored profile after rejected writes: %+v", profile)
	}
	if tiers := readTiers(); len(tiers) != 1 || tiers[0].ID != tier.ID || tiers[0].Name != "Original tier" || tiers[0].Description != "Original benefits" || tiers[0].PriceKopecks != 9900 {
		t.Fatalf("unexpected stored tiers after rejected writes: %+v", tiers)
	}

	h.expect("PUT", "/me/profile", `{"nickname":"descriptions","displayName":"Original author","description":""}`, 200)
	h.expect("PUT", tierPath, `{"name":"Original tier","description":"","priceKopecks":9900}`, 200)
	if profile := readProfile(); profile == nil || profile.Description != "" {
		t.Fatal("expected the saved profile description to be empty")
	}
	if tiers := readTiers(); len(tiers) != 1 || tiers[0].ID != tier.ID || tiers[0].Description != "" {
		t.Fatal("expected the saved tier description to be empty")
	}
	empty := testTier(t, h.expect("POST", "/me/tiers", `{"name":"Free tier","description":"","priceKopecks":0}`, 201))
	if empty.Description != "" || empty.PriceKopecks != 0 {
		t.Fatalf("unexpected created free tier: %+v", empty)
	}

	h.expect("PUT", tierPath, `{"name":"Original tier","description":"  Updated benefits  ","priceKopecks":9900}`, 200)
	for _, got := range readTiers() {
		if got.ID == tier.ID && got.Description == "Updated benefits" {
			return
		}
	}
	t.Fatal("expected the updated tier description to be trimmed")
}
