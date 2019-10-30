package main

import (
	"encoding/json"
	"fmt"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
	"io"
	"sync"
)

type Broadcaster struct {
	connections map[*connection]*connection
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{make(map[*connection]*connection)}
}

func (b *Broadcaster) removeConnection(conn *connection) {
	delete(b.connections, conn)
}

func (b *Broadcaster) Connect(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	mediaEngine := &webrtc.MediaEngine{}
	mediaEngine.RegisterDefaultCodecs()

	api := webrtc.NewAPI(webrtc.WithMediaEngine(*mediaEngine))
	peerConnection, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: iceServers,
	})
	if err != nil {
		panic(err)
	}

	videoTrack, err := peerConnection.NewTrack(webrtc.DefaultPayloadTypeVP8, 5000, "video", "pion_video")
	if err != nil {
		panic(err)
	}

	_, err = peerConnection.AddTrack(videoTrack)
	if err != nil {
		panic(err)
	}

	audioTrack, err := peerConnection.NewTrack(webrtc.DefaultPayloadTypeOpus, 5001, "audio", "pion_audio")
	if err != nil {
		panic(err)
	}

	_, err = peerConnection.AddTrack(audioTrack)
	if err != nil {
		panic(err)
	}

	connection := newConnection(peerConnection, videoTrack, audioTrack)
	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE connection state has changed %s \n", connectionState.String())
	})

	peerConnection.OnConnectionStateChange(func(connectionState webrtc.PeerConnectionState) {
		fmt.Printf("connection state has changed %s \n", connectionState.String())
		switch connectionState {
		case webrtc.PeerConnectionStateDisconnected,
			webrtc.PeerConnectionStateFailed,
			webrtc.PeerConnectionStateClosed:
			b.removeConnection(connection)
			break
		}
	})

	peerConnection.OnTrack(func(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
		println(track.Codec().Name)
		if track.Kind() == webrtc.RTPCodecTypeAudio {
			go io.Copy(audioTrack, track)
		}
		if track.Kind() == webrtc.RTPCodecTypeVideo {
			go io.Copy(videoTrack, track)
		}
	})

	// Set the remote SessionDescription
	if err = peerConnection.SetRemoteDescription(offer); err != nil {
		panic(err)
	}

	// Create answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	if err = peerConnection.SetLocalDescription(answer); err != nil {
		panic(err)
	}

	b.connections[connection] = connection
	return answer, nil
}

func (b *Broadcaster) broadcastVideo(frame []byte) {
	for _, conn := range b.connections {
		if conn.isActive() {
			_, err := conn.WriteVideo(frame)
			if err != nil {
				println("During writing video to connection, error occured: ", err.Error())
			}

		}
	}
}

func (b *Broadcaster) broadcastAudio(frame []byte) {
	for _, conn := range b.connections {
		if conn.isActive() {
			_, err := conn.WriteAudio(frame)
			if err != nil {
				println("During writing audio to connection, error occured: ", err.Error())
			}
		}
	}
}

type acceptOfferPayload struct {
	Offer webrtc.SessionDescription `json:"offer"`
}

type acceptOfferAnswer struct {
	Answer  webrtc.SessionDescription `json:"answer"`
	Success bool                      `json:"success"`
}

func (b *Broadcaster) acceptOffer(offer []byte) []byte {
	payload := &acceptOfferPayload{}
	err := json.Unmarshal(offer, payload)
	if err != nil {
		panic(err)
	}
	answer, err := b.Connect(payload.Offer)
	if err != nil {
		panic(err)
	}
	response, err := json.Marshal(acceptOfferAnswer{Answer: answer, Success: true})
	if err != nil {
		panic(err)
	}
	return response
}

type connection struct {
	peerConnection *webrtc.PeerConnection
	videoTrack     *webrtc.Track
	audioTrack     *webrtc.Track
	audioMutex     *sync.Mutex
}

func newConnection(peerConnection *webrtc.PeerConnection, videoTrack *webrtc.Track, audioTrack *webrtc.Track) *connection {
	return &connection{peerConnection: peerConnection, videoTrack: videoTrack, audioTrack: audioTrack, audioMutex: &sync.Mutex{}}
}

func (c *connection) isActive() bool {
	return c.peerConnection.ConnectionState() == webrtc.PeerConnectionStateConnected
}

func (c *connection) WriteVideo(frame []byte) (n int, err error) {
	err = c.videoTrack.WriteSample(media.Sample{Data: frame, Samples: 90000})
	if err != nil {
		return 0, err
	}
	return len(frame), nil
}
func (c *connection) WriteAudio(frame []byte) (n int, err error) {
	c.audioMutex.Lock()
	err = c.audioTrack.WriteSample(media.Sample{Data: frame, Samples: 48000})
	c.audioMutex.Unlock()
	if err != nil {
		return 0, err
	}
	return len(frame), nil
}

var iceServers = []webrtc.ICEServer{
	{
		URLs: []string{"stun:stun.l.google.com:19302"},
	},
}
