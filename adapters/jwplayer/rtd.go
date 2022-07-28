package jwplayer

import "github.com/mxmCherry/openrtb/v15/openrtb2"

type RTDAdapter interface {
	EnrichRequest(request *openrtb2.BidRequest, siteId string) EnrichmentFailed
}
