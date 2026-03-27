// Package twitch implements the Twitch platform provider for MongeBot.
package twitch

// Twitch API endpoints
const (
	GQLURL       = "https://gql.twitch.tv/gql"
	HelixURL     = "https://api.twitch.tv/helix/users"
	PubSubURL    = "wss://pubsub-edge.twitch.tv/v1"
	ChatURL      = "wss://irc-ws.chat.twitch.tv:443"
	UsherURL     = "https://usher.ttvnw.net/api/channel/hls"
	SpadeBaseURL = "https://spade.twitch.tv"

	WebURL    = "https://www.twitch.tv"
	MobileURL = "https://m.twitch.tv"
	TVURL     = "https://android.tv.twitch.tv"
)

// Twitch Client IDs for different platforms
const (
	PCClientID      = "kimne78kx3ncx6brgo4mv6wki5h1ko"
	TVClientID      = "ue6666qo983tsx6so1t0vnawi233wa"
	MobileClientID  = "r8s4dac0uhzifbpu9sjdiwzctle17ff"
	AndroidClientID = "kd1unb4b3q4t58fwlpcbzcbnm76a8fp"
	IOSClientID     = "851cqzxpb9bqu9z6galo155du"
	HelixClientID   = "d4uvtfdr04uq6raoenvj7m86gdk16v"
)

// Player configuration
const (
	PlayerType    = "pulsar"
	PlayerBackend = "mediaplayer"
	PlayerVersion = "1.22.0"
)

// PubSub ping interval
const PingInterval = 240 // 4 minutes in seconds

// GQL operation hashes
var gqlOperations = map[string]string{
	"WatchTrackQuery": "ecdcb724b0559d49689e6a32795e6a43bba4b2071b5e762a4d1edf2bb42a6789",
}

// Origins for request randomization
var origins = []string{WebURL, MobileURL, TVURL}
