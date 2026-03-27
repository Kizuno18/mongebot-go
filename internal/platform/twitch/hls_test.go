package twitch

import (
	"testing"
)

func TestParseBestStreamURL(t *testing.T) {
	playlist := `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=6000000,RESOLUTION=1920x1080,CODECS="avc1.64002A,mp4a.40.2",VIDEO="chunked"
https://video-weaver.ams02.hls.ttvnw.net/v1/playlist/best.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=2000000,RESOLUTION=1280x720,CODECS="avc1.4D401F,mp4a.40.2",VIDEO="720p30"
https://video-weaver.ams02.hls.ttvnw.net/v1/playlist/720p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=500000,RESOLUTION=640x360,CODECS="avc1.4D401E,mp4a.40.2",VIDEO="360p30"
https://video-weaver.ams02.hls.ttvnw.net/v1/playlist/360p.m3u8`

	url := parseBestStreamURL(playlist)
	if url != "https://video-weaver.ams02.hls.ttvnw.net/v1/playlist/best.m3u8" {
		t.Errorf("expected best quality URL, got %s", url)
	}
}

func TestParseBestStreamURL_Empty(t *testing.T) {
	url := parseBestStreamURL("")
	if url != "" {
		t.Errorf("expected empty for empty playlist, got %s", url)
	}
}

func TestParseLastSegmentURL_Absolute(t *testing.T) {
	playlist := `#EXTM3U
#EXT-X-MEDIA-SEQUENCE:12345
#EXTINF:2.000,
https://video-edge.ams02.abs.hls.ttvnw.net/seg1.ts
#EXTINF:2.000,
https://video-edge.ams02.abs.hls.ttvnw.net/seg2.ts`

	url := parseLastSegmentURL(playlist, "https://base.url/playlist.m3u8")
	if url != "https://video-edge.ams02.abs.hls.ttvnw.net/seg2.ts" {
		t.Errorf("expected last segment URL, got %s", url)
	}
}

func TestParseLastSegmentURL_Relative(t *testing.T) {
	playlist := `#EXTM3U
#EXTINF:2.000,
segment-001.ts
#EXTINF:2.000,
segment-002.ts`

	url := parseLastSegmentURL(playlist, "https://video.twitch.tv/v1/playlist/live.m3u8")
	if url != "https://video.twitch.tv/v1/playlist/segment-002.ts" {
		t.Errorf("expected resolved URL, got %s", url)
	}
}

func TestParseHLSAttributes(t *testing.T) {
	raw := `ID="ad-123",CLASS="twitch-stitched-ad",X-TV-TWITCH-AD-ROLL-TYPE="preroll",DURATION=30.0`
	attrs := parseHLSAttributes(raw)

	if attrs["ID"] != "ad-123" {
		t.Errorf("expected ID=ad-123, got %s", attrs["ID"])
	}
	if attrs["CLASS"] != "twitch-stitched-ad" {
		t.Errorf("expected CLASS=twitch-stitched-ad, got %s", attrs["CLASS"])
	}
	if attrs["X-TV-TWITCH-AD-ROLL-TYPE"] != "preroll" {
		t.Errorf("expected rollType=preroll, got %s", attrs["X-TV-TWITCH-AD-ROLL-TYPE"])
	}
}

func TestDetectStitchedAd(t *testing.T) {
	playlist := `#EXTM3U
#EXT-X-DATERANGE:ID="ad-456",CLASS="twitch-stitched-ad",X-TV-TWITCH-AD-ROLL-TYPE="midroll"
#EXTINF:2.0,
segment.ts`

	adID, rollType := detectStitchedAd(playlist)
	if adID != "ad-456" {
		t.Errorf("expected adID=ad-456, got %s", adID)
	}
	if rollType != "midroll" {
		t.Errorf("expected rollType=midroll, got %s", rollType)
	}
}

func TestDetectStitchedAd_NoAd(t *testing.T) {
	playlist := `#EXTM3U
#EXTINF:2.0,
segment.ts`

	adID, _ := detectStitchedAd(playlist)
	if adID != "" {
		t.Errorf("expected no ad, got %s", adID)
	}
}

func TestExtractMediaSequence(t *testing.T) {
	playlist := `#EXTM3U
#EXT-X-MEDIA-SEQUENCE:12345
#EXTINF:2.0,
segment.ts`

	seq := extractMediaSequence(playlist)
	if seq != "12345" {
		t.Errorf("expected 12345, got %s", seq)
	}
}
