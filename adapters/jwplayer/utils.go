package jwplayer

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/macros"
	"net/url"
	"strconv"
	"strings"
	"text/template"
)

func parseExtraInfo(v string) ExtraInfo {
	var extraInfo ExtraInfo
	if err := json.Unmarshal([]byte(v), &extraInfo); err != nil {
		extraInfo = ExtraInfo{}
	}

	if extraInfo.TargetingEndpoint == "" {
		extraInfo.TargetingEndpoint = "https://content-targeting-api.longtailvideo.com/property/{{.SiteId}}/content_segments?content_url=%{{.MediaUrl}}&title={{.Title}}&description={{.Description}}"
	}

	return extraInfo
}

// copied from appnexus.go appnexusImpExtAppnexus
type appnexusImpExtParams struct {
	PlacementID int `json:"placement_id,omitempty"`
}

// copied from appnexus.go appnexusImpExt
type appnexusImpExt struct {
	Appnexus appnexusImpExtParams `json:"appnexus"`
}

func GetAppnexusExt(placementId string) json.RawMessage {
	id, conversionError := strconv.Atoi(placementId)
	if conversionError != nil {
		return nil
	}

	appnexusExt := &appnexusImpExt{
		Appnexus: appnexusImpExtParams{
			PlacementID: id,
		},
	}

	jsonExt, jsonError := json.Marshal(appnexusExt)
	if jsonError != nil {
		return nil
	}

	return jsonExt
}

func ParsePublisherParams(publisher openrtb2.Publisher) *jwplayerPublisher {
	var pubExt publisherExt
	if err := json.Unmarshal(publisher.Ext, &pubExt); err != nil {
		return nil
	}

	return &pubExt.JWPlayer
}

func ParseContentMetadata(content openrtb2.Content) ContentMetadata {
	metadata := ContentMetadata{
		Url:   content.URL,
		Title: content.Title,
	}

	contentExt := ContentExt{}
	if error := json.Unmarshal(content.Ext, &contentExt); error == nil {
		println("karim: DESCRIPTION: ", contentExt.Description)
		metadata.Description = contentExt.Description
	}

	return metadata
}

func GetExistingJwpsegs(data []openrtb2.Data) []string {
	for _, datum := range data {
		if HasJwpsegs(datum) {
			return ParseJwpsegs(datum.Segment)
		}
	}

	return nil
}

func HasJwpsegs(datum openrtb2.Data) bool {
	dataExt := DataExt{}
	if error := json.Unmarshal(datum.Ext, &dataExt); error != nil {
		return false
	}

	return datum.Name == jwplayerDomain && dataExt.Segtax == jwplayerSegtax && len(datum.Segment) > 0
}

func IsValidMediaUrl(rawUrl string) bool {
	parsedUrl, error := url.ParseRequestURI(rawUrl)
	if error != nil {
		return false
	}

	isLocalFile := parsedUrl.Scheme == "file"
	isLocalHost := parsedUrl.Opaque != ""
	path := parsedUrl.Path
	isRelativePath := parsedUrl.Scheme == "" && len(path) > 2 && path[0:1] == "/" && path[1:2] != "/"

	return !isLocalFile && !isLocalHost && !isRelativePath
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

func ParseJwpsegs(segments []openrtb2.Segment) []string {
	jwpsegs := make([]string, len(segments))
	for index, segment := range segments {
		jwpsegs[index] = segment.Value
	}

	return jwpsegs
}

func GetAllJwpsegs(targeting TargetingData) []string {
	return append(targeting.BaseSegments, targeting.TargetingProfiles...)
}

func MakeOrtbDatum(jwpsegs []string) (contentData openrtb2.Data) {
	contentData.Name = jwplayerDomain
	contentData.Segment = MakeOrtbSegments(jwpsegs)
	dataExt := DataExt{
		Segtax: jwplayerSegtax,
	}
	contentData.Ext, _ = json.Marshal(dataExt)
	return contentData
}

func MakeOrtbSegments(jwpsegs []string) []openrtb2.Segment {
	segments := make([]openrtb2.Segment, len(jwpsegs))
	for index, jwpseg := range jwpsegs {
		segment := openrtb2.Segment{
			Value: jwpseg,
		}
		segments[index] = segment
	}

	return segments
}

func WriteToXandrKeywords(keywords *string, jwpsegs []string) {
	if len(jwpsegs) == 0 {
		return
	}

	if len(*keywords) > 0 {
		*keywords += ","
	}

	*keywords += ConvertToXandrKeywords(jwpsegs)
}

func ConvertToXandrKeywords(jwpsegs []string) string {
	if len(jwpsegs) == 0 {
		return ""
	}

	keyword := "jwpseg="
	// expected format: jwpseg=1,jwpseg=2,jwpseg=3
	keyword += strings.Join(jwpsegs, ","+keyword)
	return keyword
}
