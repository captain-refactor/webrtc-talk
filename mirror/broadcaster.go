package main

import (
	"encoding/json"
	"fmt"
	webrtc_talk "github.com/captain-refactor/webrtc-talk"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v2"
	"io"
)

type Broadcaster struct {
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{}
}

func (b *Broadcaster) Connect(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	mediaEngine := &webrtc.MediaEngine{}
	mediaEngine.RegisterDefaultCodecs()

	api := webrtc.NewAPI(webrtc.WithMediaEngine(*mediaEngine))
	peerConnection, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: webrtc_talk.IceServers,
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
			break
		}
	})

	peerConnection.OnTrack(func(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
		println(track.Codec().Name)
		if track.Kind() == webrtc.RTPCodecTypeAudio {
			go io.Copy(audioTrack, track)
		}
		if track.Kind() == webrtc.RTPCodecTypeVideo {
			errSend := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: track.SSRC()}})
			if errSend != nil {
				fmt.Println(errSend)
			}
			fmt.Printf("Track has started, of type %d: %s \n", track.PayloadType(), track.Codec().Name)
			for {
				// Read RTP packets being sent to Pion
				rtp, readErr := track.ReadRTP()
				if readErr != nil {
					panic(readErr)
				}

				// Replace the SSRC with the SSRC of the outbound track.
				// The only change we are making replacing the SSRC, the RTP packets are unchanged otherwise
				rtp.SSRC = videoTrack.SSRC()

				if writeErr := videoTrack.WriteRTP(rtp); writeErr != nil {
					panic(writeErr)
				}
			}
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

	return answer, nil
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
