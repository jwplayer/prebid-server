package jwplayer

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseExtraInfo(t *testing.T) {
	extraInfo := ParseExtraInfo("{\"targeting_endpoint\": \"targetingUrl\"}")
	assert.Equal(t, "targetingUrl", extraInfo.TargetingEndpoint)

	defaultTargetingUrl := "https://content-targeting-api.longtailvideo.com/property/{{.SiteId}}/content_segments?content_url=%{{.MediaUrl}}&title={{.Title}}&description={{.Description}}"
	extraInfo = ParseExtraInfo("{/")
	assert.Equal(t, defaultTargetingUrl, extraInfo.TargetingEndpoint)

	extraInfo = ParseExtraInfo("{}")
	assert.Equal(t, defaultTargetingUrl, extraInfo.TargetingEndpoint)
}

func TestGetAllJwpsegs(t *testing.T) {
	targetingData := TargetingData{
		BaseSegments:      []string{"1", "2", "3"},
		TargetingProfiles: []string{"4", "5"},
	}

	jwpsegs := GetAllJwpsegs(targetingData)
	expectedJwpsegs := []string{"1", "2", "3", "4", "5"}

	assert.Len(t, jwpsegs, len(expectedJwpsegs))
	assert.ElementsMatch(t, jwpsegs, expectedJwpsegs)
}
