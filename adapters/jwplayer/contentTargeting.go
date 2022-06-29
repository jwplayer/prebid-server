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

type Enricher interface {
	EnrichRequest(request *openrtb2.BidRequest, siteId string) *TargetingFailed
}

type RequestEnricher struct {
	httpClient       *http.Client
	EndpointTemplate *template.Template
}

func buildRequestEnricher(httpClient *http.Client, targetingEndpoint string) (*RequestEnricher, *TargetingFailed) {
	template, parseError := template.New("targetingEndpointTemplate").Parse(targetingEndpoint)
	var buildError *TargetingFailed = nil
	if parseError != nil {
		buildError = &TargetingFailed{
			Message: fmt.Sprintf("Unable to parse targeting url template: %v", parseError),
			code:    EndpointTemplateErrorCode,
		}
	}

	return &RequestEnricher{
		httpClient:       httpClient,
		EndpointTemplate: template,
	}, buildError
}

func (enricher *RequestEnricher) EnrichRequest(request *openrtb2.BidRequest, siteId string) *TargetingFailed {
	if site := request.Site; site != nil {
		return enricher.enrichFields(&site.Keywords, site.Content, siteId)
	}

	if app := request.App; app != nil {
		return enricher.enrichFields(&app.Keywords, app.Content, siteId)
	}

	return &TargetingFailed{
		Message: "Missing request.{site|app}",
		code:    MissingDistributionChannelErrorCode,
	}
}

func (enricher *RequestEnricher) enrichFields(keywords *string, content *openrtb2.Content, siteId string) *TargetingFailed {
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

	if enricher.EndpointTemplate == nil {
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

	targetingUrl := BuildTargetingEndpoint(enricher.EndpointTemplate, siteId, metadata)
	if targetingUrl == "" {
		return &TargetingFailed{
			Message: "Failed to build the targeting Url",
			code:    TargetingUrlErrorCode,
		}
	}

	channel := make(chan TargetingOutcome, 1)

	go func() {
		response, err := enricher.fetchContentTargeting(targetingUrl)
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

func (enricher *RequestEnricher) fetchContentTargeting(targetingUrl string) (*TargetingResponse, *TargetingFailed) {
	httpReq, newReqErr := http.NewRequest("GET", targetingUrl, nil)
	if newReqErr != nil {
		return nil, &TargetingFailed{
			Message: fmt.Sprintf("Failed to instantiate request: %s", newReqErr.Error()),
			code:    HttpRequestInstantiationErrorCode,
		}
	}

	resp, reqFail := enricher.httpClient.Do(httpReq)
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
