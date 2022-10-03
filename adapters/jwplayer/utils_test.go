package jwplayer

import (
	"github.com/mxmCherry/openrtb/v16/adcom1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsInstream(t *testing.T) {
	assert.True(t, IsInstream(adcom1.VideoInStream))

	assert.False(t, IsInstream(adcom1.VideoInBanner))
	assert.False(t, IsInstream(adcom1.VideoInArticle))
	assert.False(t, IsInstream(adcom1.VideoInFeed))
	assert.False(t, IsInstream(adcom1.VideoAlwaysVisible))
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
