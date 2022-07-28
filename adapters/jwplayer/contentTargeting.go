package jwplayer

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
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
	error    *TargetingFailed
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

func buildContentTargeting(httpClient *http.Client, endpoint string) (*ContentTargeting, *TargetingFailed) {
	template, parseError := template.New("targetingEndpointTemplate").Parse(endpoint)
	var buildError *TargetingFailed = nil
	if parseError != nil {
		buildError = &TargetingFailed{
			Message: fmt.Sprintf("Unable to parse targeting url template: %v", parseError),
			code:    EndpointTemplateErrorCode,
		}
	}

	return &ContentTargeting{
		httpClient:       httpClient,
		EndpointTemplate: template,
	}, buildError
}

func (ct *ContentTargeting) EnrichRequest(request *openrtb2.BidRequest, siteId string) EnrichmentFailed {
	if site := request.Site; site != nil {
		return ct.enrichFields(&site.Keywords, site.Content, siteId)
	}

	if app := request.App; app != nil {
		return ct.enrichFields(&app.Keywords, app.Content, siteId)
	}

	return &TargetingFailed{
		Message: "Missing request.{site|app}",
		code:    MissingDistributionChannelErrorCode,
	}
}

func (ct *ContentTargeting) enrichFields(keywords *string, content *openrtb2.Content, siteId string) *TargetingFailed {
	if content == nil {
		return &TargetingFailed{
			Message: "Missing request.{site|app}.content",
			code:    MissingContentBlockErrorCode,
		}
	}

	jwpsegs := GetExistingJwpsegs(content.Data)
	if len(jwpsegs) > 0 {
		WriteToXandrKeywords(keywords, jwpsegs)
		return nil
	}

	if siteId == "" {
		return &TargetingFailed{
			Message: "Missing publisher.ext.jwplayer.SiteId",
			code:    MissingSiteIdErrorCode,
		}
	}

	if ct.EndpointTemplate == nil {
		return &TargetingFailed{
			Message: "Empty template",
			code:    EmptyTemplateErrorCode,
		}
	}

	metadata := ParseContentMetadata(*content)
	if IsValidMediaUrl(metadata.Url) == false {
		return &TargetingFailed{
			Message: "Invalid Media Url",
			code:    MissingMediaUrlErrorCode,
		}
	}

	targetingUrl := BuildTargetingEndpoint(ct.EndpointTemplate, siteId, metadata)
	if targetingUrl == "" {
		return &TargetingFailed{
			Message: "Failed to build the targeting Url",
			code:    TargetingUrlErrorCode,
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
		return &TargetingFailed{
			Message: "Empty Targeting Segments",
			code:    EmptyTargetingSegmentsErrorCode,
		}
	}

	WriteToXandrKeywords(keywords, jwpsegs)

	contentDatum := MakeOrtbDatum(jwpsegs)
	content.Data = append(content.Data, contentDatum)

	return nil
}

func (ct *ContentTargeting) fetch(targetingUrl string) (*TargetingResponse, *TargetingFailed) {
	httpReq, newReqErr := http.NewRequest("GET", targetingUrl, nil)
	if newReqErr != nil {
		return nil, &TargetingFailed{
			Message: fmt.Sprintf("Failed to instantiate request: %s", newReqErr.Error()),
			code:    HttpRequestInstantiationErrorCode,
		}
	}

	resp, reqFail := ct.httpClient.Do(httpReq)
	if reqFail != nil {
		return nil, &TargetingFailed{
			Message: fmt.Sprintf("Request Execution failure: %s", reqFail.Error()),
			code:    HttpRequestExecutionErrorCode,
		}
	}

	statusCode := resp.StatusCode
	if statusCode != http.StatusOK {
		return nil, &TargetingFailed{
			Message: fmt.Sprintf("Server responded with failure status: %d.", statusCode),
			code:    BaseNetworkErrorCode + statusCode,
		}
	}

	defer resp.Body.Close()

	targetingResponse := TargetingResponse{}
	if error := json.NewDecoder(resp.Body).Decode(&targetingResponse); error != nil {
		return nil, &TargetingFailed{
			Message: fmt.Sprintf("Failed to decode targeting response: %s", error.Error()),
			code:    BaseDecodingErrorCode,
		}
	}

	return &targetingResponse, nil
}
