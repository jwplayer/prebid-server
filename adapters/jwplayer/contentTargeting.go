package jwplayer

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/macros"
	"net/http"
	"net/url"
	"text/template"
)

const jwplayerSegtax = 502
const jwplayerDomain = "jwplayer.com"

type jwContentMetadata struct {
	Url         string
	Title       string
	Description string
}

type jwContentExt struct {
	Description string `json:"description"`
}

type jwDataExt struct {
	Segtax int `json:"segtax"`
}

type enrichment struct {
	response *jwTargetingResponse
	error    *TargetingFailed
}

type jwTargetingResponse struct {
	Uuid string          `json:"uuid"`
	Data jwTargetingData `json:"data"`
}

type jwTargetingData struct {
	MediaId           string   `json:"media_id"`
	BaseSegments      []string `json:"base_segments"`
	TargetingProfiles []string `json:"targeting_profiles"`
}

type requestEnricher struct {
	httpClient       *http.Client
	EndpointTemplate *template.Template
}

type EndpointTemplateParams struct {
	SiteId      string
	MediaUrl    string
	Title       string
	Description string
}

func buildRequestEnricher(httpClient *http.Client, targetingEndpoint string) (*requestEnricher, *TargetingFailed) {
	template, err := template.New("targetingEndpointTemplate").Parse(targetingEndpoint)

	if err != nil {
		return nil, &TargetingFailed{
			Message: fmt.Sprintf("unable to parse targeting url template: %v", err),
			code:    EndpointTemplateErrorCode,
		}
	}

	return &requestEnricher{
		httpClient:       httpClient,
		EndpointTemplate: template,
	}, nil
}

func (enricher *requestEnricher) EnrichRequest(request *openrtb2.BidRequest, publisherId string) *TargetingFailed {
	if site := request.Site; site != nil {
		return enricher.enrich(&site.Keywords, site.Content, publisherId, request.ID)
	}

	if app := request.App; app != nil {
		return enricher.enrich(&app.Keywords, app.Content, publisherId, request.ID)
	}

	return nil
}

func (enricher *requestEnricher) enrich(keywords *string, content *openrtb2.Content, publisherId string, id string) *TargetingFailed {
	jwpsegs := GetExistingJwpsegs(content.Data)
	if jwpsegs != nil && len(jwpsegs) > 0 {
		writeToKeywords(keywords, jwpsegs)
		return nil
	}

	if publisherId == "" {
		return &TargetingFailed{
			Message: "Missing PublisherId",
			code:    MissingPublisherIdErrorCode,
		}
	}

	metadata := ParseContentMetadata(*content)
	if isValidMediaUrl(metadata.Url) == false {
		return &TargetingFailed{
			Message: "Missing Media Url",
			code:    MissingMediaUrlErrorCode,
		}
	}

	channel := make(chan enrichment, 1)

	fmt.Println("before go: ", id)
	go func() {
		fmt.Println("start go: ", id)
		response, err := enricher.FetchContentTargeting(publisherId, metadata)
		fmt.Println("before chann: ", id)
		channel <- enrichment{
			response: response,
			error:    err,
		}
		fmt.Println("end go: ", id)
	}()
	fmt.Println("after go: ", id)

	enrichmentResult := <-channel
	fmt.Println("after channel: ", id)

	if enrichmentResult.error != nil {
		return enrichmentResult.error
	}

	targetingResponse := enrichmentResult.response
	jwpsegs = GetAllJwpsegs(targetingResponse.Data)
	if len(jwpsegs) == 0 {
		return &TargetingFailed{
			Message: "Empty Targeting Segments",
			code:    EmptyTargetingSegments,
		}
	}

	writeToKeywords(keywords, jwpsegs)

	contentDatum := MakeOrtbDatum(jwpsegs)
	content.Data = append(content.Data, contentDatum)

	return nil
}

func (enricher *requestEnricher) FetchContentTargeting(publisherId string, contentMetadata jwContentMetadata) (*jwTargetingResponse, *TargetingFailed) {
	mediaUrl := url.QueryEscape(contentMetadata.Url)
	title := url.QueryEscape(contentMetadata.Title)
	description := url.QueryEscape(contentMetadata.Description)

	endpointParams := EndpointTemplateParams{
		SiteId:      publisherId,
		MediaUrl:    mediaUrl,
		Title:       title,
		Description: description,
	}

	reqUrl, macroResolveErr := macros.ResolveMacros(enricher.EndpointTemplate, endpointParams)
	if macroResolveErr != nil {
		return nil, &TargetingFailed{
			Message: "Failed to insert macros into targeting Url",
			code:    MacroResolveErrorCode,
		}
	}
	
	httpReq, newReqErr := http.NewRequest("GET", reqUrl, nil)
	if newReqErr != nil {
		return nil, &TargetingFailed{
			Message: fmt.Sprintf("Failed to instantiate request: %s", newReqErr.Error()),
			code:    HttpRequestInstantiationErrorCode,
		}
	}

	resp, reqErr := enricher.httpClient.Do(httpReq)

	if reqErr != nil {
		statusCode := resp.StatusCode
		return nil, &TargetingFailed{
			Message: fmt.Sprintf("Server responded with failure status: %d.", statusCode),
			code:    BaseNetworkErrorCode + statusCode,
		}
	}

	defer resp.Body.Close()

	targetingResponse := jwTargetingResponse{}
	if error := json.NewDecoder(resp.Body).Decode(&targetingResponse); error != nil {
		return nil, &TargetingFailed{
			Message: fmt.Sprintf("Failed to decode targeting response: %s", error.Error()),
			code:    BaseDecodingErrorCode,
		}
	}

	fmt.Println("targeting response: ", targetingResponse)
	return &targetingResponse, nil
}
