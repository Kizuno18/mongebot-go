package twitch

import (
	"strings"
	"testing"
)

var benchPlaylist = `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=6000000,RESOLUTION=1920x1080,CODECS="avc1.64002A,mp4a.40.2",VIDEO="chunked"
https://video-weaver.ams02.hls.ttvnw.net/v1/playlist/best.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=2000000,RESOLUTION=1280x720,CODECS="avc1.4D401F,mp4a.40.2",VIDEO="720p30"
https://video-weaver.ams02.hls.ttvnw.net/v1/playlist/720p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=500000,RESOLUTION=640x360,CODECS="avc1.4D401E,mp4a.40.2",VIDEO="360p30"
https://video-weaver.ams02.hls.ttvnw.net/v1/playlist/360p.m3u8`

func BenchmarkParseBestStreamURL(b *testing.B) {
	for b.Loop() {
		parseBestStreamURL(benchPlaylist)
	}
}

var benchSegmentPlaylist = strings.Repeat(`#EXTINF:2.000,
https://video-edge.ams02.abs.hls.ttvnw.net/v1/segment/1234.ts
`, 20)

func BenchmarkParseLastSegmentURL(b *testing.B) {
	for b.Loop() {
		parseLastSegmentURL(benchSegmentPlaylist, "https://base.url/playlist.m3u8")
	}
}

var benchDateRange = `ID="ad-12345",CLASS="twitch-stitched-ad",X-TV-TWITCH-AD-ROLL-TYPE="midroll",DURATION=30.0,START-DATE="2026-01-01T00:00:00Z"`

func BenchmarkParseHLSAttributes(b *testing.B) {
	for b.Loop() {
		parseHLSAttributes(benchDateRange)
	}
}

func BenchmarkDetectStitchedAd(b *testing.B) {
	playlist := `#EXTM3U
#EXT-X-MEDIA-SEQUENCE:12345
#EXT-X-DATERANGE:ID="ad-456",CLASS="twitch-stitched-ad",X-TV-TWITCH-AD-ROLL-TYPE="midroll"
#EXTINF:2.0,
segment.ts
#EXTINF:2.0,
segment2.ts`

	for b.Loop() {
		detectStitchedAd(playlist)
	}
}
