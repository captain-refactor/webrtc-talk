package webrtc_talk

import "github.com/pion/webrtc/v2"

var IceServers = []webrtc.ICEServer{
	{
		URLs: []string{"stun:stun.l.google.com:19302"},
	},
}
