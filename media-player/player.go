package main

import (
	"encoding/json"
	"fmt"
	"github.com/3d0c/gmf"
	"github.com/captain-refactor/webrtc-talk"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
	"io"
	"time"
)

type Player struct {
}

func NewPlayer() *Player {
	return &Player{}
}

func (p Player) Connect(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {

	mediaEngine := &webrtc.MediaEngine{}
	mediaEngine.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
	mediaEngine.RegisterCodec(webrtc.NewRTPCodec(webrtc.RTPCodecTypeAudio,
		webrtc.Opus,
		48000,
		2, //According to RFC7587, Opus RTP streams must have exactly 2 channels.
		"ptime=20",
		webrtc.DefaultPayloadTypeOpus,
		&codecs.OpusPayloader{}))

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
	sendVideo := videoSender(videoTrack)

	audioTrack, err := peerConnection.NewTrack(webrtc.DefaultPayloadTypeOpus, 5001, "audio", "pion_audio")
	if err != nil {
		panic(err)
	}

	_, err = peerConnection.AddTrack(audioTrack)
	if err != nil {
		panic(err)
	}
	sendAudio := audioSender(audioTrack)

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
		case webrtc.PeerConnectionStateConnected:
			go broadcastLoop(sendVideo, sendAudio)
			break
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

func (p Player) AcceptOffer(offer []byte) []byte {
	payload := &acceptOfferPayload{}
	err := json.Unmarshal(offer, payload)
	if err != nil {
		panic(err)
	}
	answer, err := p.Connect(payload.Offer)
	if err != nil {
		panic(err)
	}
	response, err := json.Marshal(acceptOfferAnswer{Answer: answer, Success: true})
	if err != nil {
		panic(err)
	}
	return response
}

type acceptOfferPayload struct {
	Offer webrtc.SessionDescription `json:"offer"`
}

type acceptOfferAnswer struct {
	Answer  webrtc.SessionDescription `json:"answer"`
	Success bool                      `json:"success"`
}

func getVideoStream(ctx *gmf.FmtCtx) *gmf.Stream {
	for i := 0; i < ctx.StreamsCnt(); i++ {
		stream, err := ctx.GetStream(i)
		if err != nil {
			panic(err)
		}
		if stream.IsVideo() {
			return stream
		}
	}
	return nil
}

func getAudioStream(ctx *gmf.FmtCtx) *gmf.Stream {
	for i := 0; i < ctx.StreamsCnt(); i++ {
		stream, err := ctx.GetStream(i)
		if err != nil {
			panic(err)
		}
		if stream.IsAudio() {
			return stream
		}
	}
	return nil
}

func videoSender(videoTrack *webrtc.Track) func(*gmf.Packet) {
	packets := make(chan *gmf.Packet, 10)
	go func() {
		for {
			pkt := <-packets
			err := videoTrack.WriteSample(media.Sample{Data: pkt.Data(), Samples: 90000})
			if err != nil {
				panic(err)
			}
			time.Sleep(time.Millisecond * time.Duration(pkt.Duration()))
		}
	}()
	return func(pkt *gmf.Packet) {
		packets <- pkt
	}
}

func audioSender(audioTrack *webrtc.Track) func(*gmf.Packet) {
	packets := make(chan *gmf.Packet, 10)
	go func() {
		for {
			pkt := <-packets
			err := audioTrack.WriteSample(media.Sample{Data: pkt.Data(), Samples: 48000})
			if err != nil {
				panic(err)
			}
			duration := pkt.Duration()
			time.Sleep(time.Millisecond * time.Duration(duration))
		}
	}()
	return func(pkt *gmf.Packet) {
		packets <- pkt
	}
}

func broadcastLoop(sendVideo, sendAudio func(*gmf.Packet)) {
	for {
		broadcastMedia(sendVideo, sendAudio)
	}
}

func broadcastMedia(sendVideo, sendAudio func(*gmf.Packet)) {
	filename := "media-player/sample.mkv"
	println("playing", filename)
	input, err := gmf.NewInputCtx(filename)
	if err != nil {
		panic(err)
	}
	defer input.Close()
	videoStream := getVideoStream(input)
	audioStream := getAudioStream(input)
	for {
		pkt, err := input.GetNextPacket()
		if err != nil {
			if err != io.EOF {
				panic(err)
			}
			println("End of file")
			return
		}

		switch pkt.StreamIndex() {
		case videoStream.Index():
			sendVideo(pkt)
			break
		case audioStream.Index():
			sendAudio(pkt)
			break
		}
	}
}
