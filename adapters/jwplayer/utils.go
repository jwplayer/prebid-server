package jwplayer

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
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

func ParseBidderParams(imp openrtb2.Imp) (*openrtb_ext.ImpExtJWPlayer, error) {
	var impExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &impExt); err != nil {
		return nil, err
	}

	var params *openrtb_ext.ImpExtJWPlayer
	if err := json.Unmarshal(impExt.Bidder, &params); err != nil {
		return nil, err
	}

	if params.PlacementId == "" {
		return nil, &errortypes.BadInput{
			Message: "Empty ext.prebid.bidder.jwplayer.placementId",
		}
	}

	return params, nil
}

func IsOutstream(placementType openrtb2.VideoPlacementType) bool {
	return placementType > 1
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

func MakeSChain(publisherId string, requestId string, publisherSChain *openrtb_ext.ExtRequestPrebidSChainSChain) openrtb_ext.ExtRequestPrebidSChainSChain {
	node := MakeSChainNode(publisherId, requestId)
	isComplete := 1
	var nodes []*openrtb_ext.ExtRequestPrebidSChainSChainNode
	if publisherSChain != nil {
		isComplete = publisherSChain.Complete
		nodes = publisherSChain.Nodes
	}

	nodes = append(nodes, &node)

	return openrtb_ext.ExtRequestPrebidSChainSChain{
		Ver:      "1.0",
		Complete: isComplete,
		Nodes:    nodes,
	}
}

// GetPublisherSChain25 Get the SChain from the 2.5 oRTB spec
func GetPublisherSChain25(source *openrtb2.Source) *openrtb_ext.ExtRequestPrebidSChainSChain {
	if source == nil {
		return nil
	}

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
