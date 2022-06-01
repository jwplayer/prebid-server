package jwplayer

import (
	"encoding/json"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type JWPlayerAdapter struct {
	endpoint string
}

// Builder builds a new instance of the JWPlayer adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &JWPlayerAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *JWPlayerAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
    requestCopy := *request
    for _,imp:= range requestCopy.Imp {
        placementId := imp.Ext.prebid.bidder.jwplayer.placementId
        imp.tagid = placementId
        delete(imp, "ext")
    }

    if site := requestCopy.Site; site != nil {
        delete(site, "id")

        if publisher := site.Publisher; publisher != nil {
            delete(publisher, "id")
        }
    }

    if app := requestCopy.App; app != nil {
        delete(app, "id")
    }

    requestJSON, err := json.Marshal(requestCopy)
    if err != nil {
        return nil, []error{err}
    }

    headers := http.Header{}
    headers.Add("Content-Type", "application/json;charset=utf-8")
    headers.Add("Accept", "application/json")

    requestData := &adapters.RequestData{
        Method:  "POST",
        Uri:     a.endpoint,
        Body:    requestJSON,
        Headers: headers,
    }

    return []*adapters.RequestData{requestData}, nil
}

func (a *JWPlayerAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
    return nil, nil
}
