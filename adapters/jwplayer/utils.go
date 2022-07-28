package jwplayer

import (
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"net/url"
)

func IsInstream(placementType openrtb2.VideoPlacementType) bool {
	return placementType == openrtb2.VideoPlacementTypeInStream
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

func Map[T any, M any](inputs []T, f func(T) M) []M {
	outputs := make([]M, len(inputs))
	for i, element := range inputs {
		outputs[i] = f(element)
	}
	return outputs
}
