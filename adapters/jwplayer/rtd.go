package jwplayer

import "github.com/mxmCherry/openrtb/v16/openrtb2"

type RTDAdapter interface {
	EnrichRequest(request *openrtb2.BidRequest, siteId string) *Warning
}
