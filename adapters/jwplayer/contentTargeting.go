package jwplayer

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/errortypes"
	"net/http"
	"text/template"
)

const jwplayerSegtax = 502
const jwplayerDomain = "jwplayer.com"

type ContentMetadata struct {
	Url         string
	Title       string
	Description string
}

type ContentExt struct {
	Description string `json:"description"`
}

type DataExt struct {
	Segtax int `json:"segtax"`
}

type TargetingOutcome struct {
	response *TargetingResponse
	error    *errortypes.TroubleShootingSuggestion
}

type TargetingResponse struct {
	Uuid string        `json:"uuid"`
	Data TargetingData `json:"data"`
}

type TargetingData struct {
	MediaId           string   `json:"media_id"`
	BaseSegments      []string `json:"base_segments"`
	TargetingProfiles []string `json:"targeting_profiles"`
}

type ContentTargeting struct {
	httpClient       *http.Client
	EndpointTemplate *template.Template
}

func buildContentTargeting(httpClient *http.Client, endpoint string) (*ContentTargeting, *errortypes.TroubleShootingSuggestion) {
	template, parseError := template.New("targetingEndpointTemplate").Parse(endpoint)
	var buildError *errortypes.TroubleShootingSuggestion = nil
	if parseError != nil {
		buildError = &errortypes.TroubleShootingSuggestion{
			Message: fmt.Sprintf("Unable to parse targeting url template: %v", parseError),
		}
	}

	return &ContentTargeting{
		httpClient:       httpClient,
		EndpointTemplate: template,
	}, buildError
}

func (ct *ContentTargeting) EnrichRequest(request *openrtb2.BidRequest, siteId string) *errortypes.TroubleShootingSuggestion {
	if site := request.Site; site != nil {
		return ct.enrichFields(&site.Keywords, site.Content, siteId)
	}

	if app := request.App; app != nil {
		return ct.enrichFields(&app.Keywords, app.Content, siteId)
	}

	return &errortypes.TroubleShootingSuggestion{
		Message: "Missing request.{site|app}",
	}
}

func (ct *ContentTargeting) enrichFields(keywords *string, content *openrtb2.Content, siteId string) *errortypes.TroubleShootingSuggestion {
	if content == nil {
		return &errortypes.TroubleShootingSuggestion{
			Message: "Missing request.{site|app}.content",
		}
	}

	jwpsegs := GetExistingJwpsegs(content.Data)
	if len(jwpsegs) > 0 {
		WriteToXandrKeywords(keywords, jwpsegs)
		return nil
	}

	if siteId == "" {
		return &errortypes.TroubleShootingSuggestion{
			Message: "Missing publisher.ext.jwplayer.SiteId",
		}
	}

	if ct.EndpointTemplate == nil {
		return &errortypes.TroubleShootingSuggestion{
			Message: "Empty template",
		}
	}

	metadata := ParseContentMetadata(*content)
	if IsValidMediaUrl(metadata.Url) == false {
		return &errortypes.TroubleShootingSuggestion{
			Message: "Invalid Media Url",
		}
	}

	targetingUrl := BuildTargetingEndpoint(ct.EndpointTemplate, siteId, metadata)
	if targetingUrl == "" {
		return &errortypes.TroubleShootingSuggestion{
			Message: "Failed to build the targeting Url",
		}
	}

	channel := make(chan TargetingOutcome, 1)

	go func() {
		response, err := ct.fetch(targetingUrl)
		channel <- TargetingOutcome{
			response: response,
			error:    err,
		}
	}()

	targetingOutcome := <-channel

	if targetingOutcome.error != nil {
		return targetingOutcome.error
	}

	targetingResponse := targetingOutcome.response
	jwpsegs = GetAllJwpsegs(targetingResponse.Data)
	if len(jwpsegs) == 0 {
		return &errortypes.TroubleShootingSuggestion{
			Message: "Empty Targeting Segments",
		}
	}

	WriteToXandrKeywords(keywords, jwpsegs)

	contentDatum := MakeOrtbDatum(jwpsegs)
	content.Data = append(content.Data, contentDatum)

	return nil
}

func (ct *ContentTargeting) fetch(targetingUrl string) (*TargetingResponse, *errortypes.TroubleShootingSuggestion) {
	httpReq, newReqErr := http.NewRequest("GET", targetingUrl, nil)
	if newReqErr != nil {
		return nil, &errortypes.TroubleShootingSuggestion{
			Message: fmt.Sprintf("Failed to instantiate request: %s", newReqErr.Error()),
		}
	}

	resp, reqFail := ct.httpClient.Do(httpReq)
	if reqFail != nil {
		return nil, &errortypes.TroubleShootingSuggestion{
			Message: fmt.Sprintf("Request Execution failure: %s", reqFail.Error()),
		}
	}

	statusCode := resp.StatusCode
	if statusCode != http.StatusOK {
		return nil, &errortypes.TroubleShootingSuggestion{
			Message: fmt.Sprintf("Server responded with failure status: %d.", statusCode),
		}
	}

	defer resp.Body.Close()

	targetingResponse := TargetingResponse{}
	if error := json.NewDecoder(resp.Body).Decode(&targetingResponse); error != nil {
		return nil, &errortypes.TroubleShootingSuggestion{
			Message: fmt.Sprintf("Failed to decode targeting response: %s", error.Error()),
		}
	}

	return &targetingResponse, nil
}
