package jwplayer

import (
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsInstream(t *testing.T) {
	assert.True(t, IsInstream(adcom1.VideoPlacementInStream))

	assert.False(t, IsInstream(adcom1.VideoPlacementInBanner))
	assert.False(t, IsInstream(adcom1.VideoPlacementInArticle))
	assert.False(t, IsInstream(adcom1.VideoPlacementInFeed))
	assert.False(t, IsInstream(adcom1.VideoPlacementAlwaysVisible))
}

func TestIsValidMediaUrl(t *testing.T) {
	assert.False(t, IsValidMediaUrl(""))
	assert.False(t, IsValidMediaUrl("nothing"))
	assert.False(t, IsValidMediaUrl("media.mp4"))
	assert.False(t, IsValidMediaUrl("file://hello.com/media.mp4"))
	assert.False(t, IsValidMediaUrl("localhost:9999/hello.com/media.mp4"))
	assert.False(t, IsValidMediaUrl("0.0.0.0:9999/hello.com/media.mp4"))
	assert.False(t, IsValidMediaUrl("/hello.com/media.mp4"))
	assert.False(t, IsValidMediaUrl("www.example.com/video.mp4"))

	assert.True(t, IsValidMediaUrl("//hello.com/media.mp4"))
	assert.True(t, IsValidMediaUrl("http://hello.com/media.mp4"))
	assert.True(t, IsValidMediaUrl("https://hello.com/media.mp4"))
	assert.True(t, IsValidMediaUrl("https://hello.com/media.mp4?additional=sthg"))
}
