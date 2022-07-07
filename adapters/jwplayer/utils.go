package jwplayer

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/url"
	"strconv"
	"strings"
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

	if isLocalFile || isLocalHost {
		return false
	}

	path := parsedUrl.Path
	isRelativePath := parsedUrl.Scheme == "" && len(path) > 2 && path[0:1] == "/" && path[1:2] != "/"

	return !isRelativePath
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

func ParseJwpsegs(segments []openrtb2.Segment) []string {
	segmentToJwpseg := func(segment openrtb2.Segment) string { return segment.Value }
	jwpsegs := Map(segments, segmentToJwpseg)
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
	jwpsegToSegment := func(jwpseg string) openrtb2.Segment { return openrtb2.Segment{Value: jwpseg} }
	segments := Map(jwpsegs, jwpsegToSegment)
	return segments
}

func WriteToXandrKeywords(keywords *string, jwpsegs []string) {
	if len(jwpsegs) == 0 {
		return
	}

	if len(*keywords) > 0 {
		*keywords += ","
	}

	jwpsegToKeyword := func(jwpseg string) string { return "jwpseg=" + jwpseg }
	newKeywords := Map(jwpsegs, jwpsegToKeyword)
	*keywords += strings.Join(newKeywords, ",")
}

func Map[T any, M any](inputs []T, f func(T) M) []M {
	outputs := make([]M, len(inputs))
	for i, element := range inputs {
		outputs[i] = f(element)
	}
	return outputs
}

type requestExt struct {
	SChain openrtb_ext.ExtRequestPrebidSChainSChain `json:"schain"`
}

func MakeSChain(request *openrtb2.BidRequest, publisherId string) openrtb_ext.ExtRequestPrebidSChainSChain {
	node := MakeSChainNode(publisherId, request.ID)
	pub25SChain := GetPublisherSChain25(*request.Source)
	isComplete := 1
	var nodes []*openrtb_ext.ExtRequestPrebidSChainSChainNode
	if pub25SChain != nil {
		isComplete = pub25SChain.Complete
		nodes = pub25SChain.Nodes
	}

	nodes = append(nodes, &node)

	return openrtb_ext.ExtRequestPrebidSChainSChain{
		Ver:      "1.0",
		Complete: isComplete,
		Nodes:    nodes,
	}
}

// GetPublisherSChain25 Get the SChain from the 2.5 oRTB spec
func GetPublisherSChain25(source openrtb2.Source) *openrtb_ext.ExtRequestPrebidSChainSChain {
	var sourceExt openrtb_ext.SourceExt
	if err := json.Unmarshal(source.Ext, &sourceExt); err != nil {
		return nil
	}

	return &sourceExt.SChain
}

func MakeSChainNode(publisherId string, requestId string) openrtb_ext.ExtRequestPrebidSChainSChainNode {
	return openrtb_ext.ExtRequestPrebidSChainSChainNode{
		ASI: jwplayerDomain,
		SID: publisherId,
		RID: requestId,
		HP:  1,
	}
}

func GetXandrRequestExt(schain openrtb_ext.ExtRequestPrebidSChainSChain) json.RawMessage {
	// Xandr expects the SChain to be in accordance with oRTB 2.4
	// $.ext.schain
	requestExtension := requestExt{
		SChain: schain,
	}
	jsonExt, jsonError := json.Marshal(requestExtension)
	if jsonError != nil {
		return nil
	}
	return jsonExt
}
