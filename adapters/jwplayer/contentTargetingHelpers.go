package jwplayer

import (
	"encoding/json"
	"github.com/prebid/prebid-server/v3/macros"
	"net/url"
	"text/template"
)

func ParseExtraInfo(v string) ExtraInfo {
	var extraInfo ExtraInfo
	if err := json.Unmarshal([]byte(v), &extraInfo); err != nil {
		extraInfo = ExtraInfo{}
	}

	if extraInfo.TargetingEndpoint == "" {
		extraInfo.TargetingEndpoint = "https://content-targeting-api.longtailvideo.com/property/{{.SiteId}}/content_segments?content_url=%{{.MediaUrl}}&title={{.Title}}&description={{.Description}}"
	}

	return extraInfo
}

type EndpointTemplateParams struct {
	SiteId      string
	MediaUrl    string
	Title       string
	Description string
}

func BuildTargetingEndpoint(endpointTemplate *template.Template, siteId string, contentMetadata ContentMetadata) string {
	if endpointTemplate == nil {
		return ""
	}

	mediaUrl := url.QueryEscape(contentMetadata.Url)
	title := url.QueryEscape(contentMetadata.Title)
	description := url.QueryEscape(contentMetadata.Description)

	endpointParams := EndpointTemplateParams{
		SiteId:      siteId,
		MediaUrl:    mediaUrl,
		Title:       title,
		Description: description,
	}

	reqUrl, macroResolveErr := macros.ResolveMacros(endpointTemplate, endpointParams)
	if macroResolveErr != nil {
		return ""
	}

	return reqUrl
}

func GetAllJwpsegs(targeting TargetingData) []string {
	return append(targeting.BaseSegments, targeting.TargetingProfiles...)
}
