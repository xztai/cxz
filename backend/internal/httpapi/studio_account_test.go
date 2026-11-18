package httpapi

import (
	"bytes"
	"encoding/json/v2"
	"image"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestIntegrationStudioExpectedAccount(t *testing.T) {
	h := setup(t)
	objects := &memoryImages{objects: make(map[string][]byte)}
	h.server.images = objects

	request := func(method, path, contentType string, body io.Reader, csrf bool, status int, expected ...string) []byte {
		t.Helper()
		req, err := http.NewRequestWithContext(t.Context(), method, h.url+"/api/v1"+path, body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", contentType)
		if csrf {
			req.Header.Set("Origin", "http://example.test")
			req.Header.Set("X-CSRF-Token", h.csrf)
		}
		for _, value := range expected {
			req.Header.Add(expectedUserHeader, value)
		}
		resp, err := h.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != status {
			t.Fatalf("%s %s: got status %d, want %d", method, path, resp.StatusCode, status)
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if status == http.StatusPreconditionFailed && !bytes.Contains(data, []byte(`"code":"account_changed"`)) {
			t.Fatal("missing account_changed code")
		}
		return data
	}

	type ownProfileResponse struct {
		OwnerID string   `json:"ownerId"`
		Profile *Profile `json:"profile"`
	}
	parseProfile := func(data []byte) ownProfileResponse {
		t.Helper()
		var out ownProfileResponse
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatal("invalid profile response", err)
		}
		if out.OwnerID == "" {
			t.Fatal("missing profile owner")
		}
		return out
	}

	const profilePath = "/me/profile"
	const avatarPath = "/me/profile/images/avatar"
	const firstProfile = `{"nickname":"painter","displayName":"Художник","description":""}`
	const secondProfile = `{"nickname":"potter","displayName":"Керамист","description":""}`
	request("GET", profilePath, "", nil, false, http.StatusUnauthorized, "1")
	newTestSession(t, h, "painter")
	first := parseProfile(request("GET", profilePath, "", nil, false, http.StatusOK))
	if first.Profile != nil {
		t.Fatal("expected no profile before first save")
	}
	request("PUT", profilePath, "application/json", strings.NewReader(firstProfile), false, http.StatusForbidden, first.OwnerID)
	saved := parseProfile(request("PUT", profilePath, "application/json", strings.NewReader(firstProfile), true, http.StatusOK, first.OwnerID))
	if saved.OwnerID != first.OwnerID || saved.Profile == nil || saved.Profile.ID != first.OwnerID {
		t.Fatal("saved profile has the wrong owner")
	}

	newTestSession(t, h, "potter")
	second := parseProfile(request("GET", profilePath, "", nil, false, http.StatusOK))
	if second.OwnerID == first.OwnerID || second.Profile != nil {
		t.Fatal("expected a separate account with no profile")
	}
	request("PUT", profilePath, "application/json", strings.NewReader(secondProfile), true, http.StatusOK)
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, image.NewNRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	uploaded := parseProfile(request("PUT", avatarPath, "image/png", &pngData, true, http.StatusOK, second.OwnerID))
	if uploaded.OwnerID != second.OwnerID || uploaded.Profile == nil || uploaded.Profile.Avatar == nil {
		t.Fatal("uploaded image is missing from owner's profile")
	}
	const staleDraft = `{"nickname":"potter","displayName":"Устаревший черновик","description":"Не сохранять"}`
	request("PUT", profilePath, "application/json", strings.NewReader(staleDraft), true, http.StatusPreconditionFailed, first.OwnerID)
	request("DELETE", avatarPath, "", nil, true, http.StatusPreconditionFailed, first.OwnerID)
	for _, malformed := range [][]string{{""}, {"0" + second.OwnerID}, {second.OwnerID + ", " + second.OwnerID}, {second.OwnerID, second.OwnerID}} {
		request("GET", profilePath, "", nil, false, http.StatusPreconditionFailed, malformed...)
	}
	unchanged := parseProfile(request("GET", profilePath, "", nil, false, http.StatusOK))
	if unchanged.OwnerID != second.OwnerID || unchanged.Profile == nil ||
		unchanged.Profile.DisplayName != uploaded.Profile.DisplayName ||
		unchanged.Profile.Description != uploaded.Profile.Description ||
		unchanged.Profile.Avatar == nil || *unchanged.Profile.Avatar != *uploaded.Profile.Avatar {
		t.Fatal("rejected mutation changed the current account's profile")
	}
	request("GET", "/authors/potter/images/avatar", "", nil, false, http.StatusOK)
}
