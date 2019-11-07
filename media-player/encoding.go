package main

import "github.com/3d0c/gmf"

type convertService struct {
	ctx   *gmf.FmtCtx
	audio *codecService
	video *codecService
}

func newConvertService() *convertService {
	ctx := gmf.NewCtx()
	return &convertService{
		ctx:   ctx,
		audio: NewCodecService(ctx, "libopus"),
		video: NewCodecService(ctx, "libvpx"),
	}
}

func (c convertService) Encode(input *gmf.FmtCtx) {
	inputAudio := getAudioStream(input)
	inputVideo := getVideoStream(input)

	for {
		pkt, err := input.GetNextPacket()
		if err != nil {
			panic(err)
		}
		switch pkt.StreamIndex() {
		case inputAudio.Index():
			frames, err := inputAudio.CodecCtx().Decode(pkt)
			if err != nil {
				panic(err)
			}
			encoded, err := c.audio.codecCtx.Encode(frames, 0)
			if err != nil {
				panic(err)
			}
			//TODO
			break
		case inputVideo.Index():
			break
		}

	}
}

func (c convertService) Release() {
	c.video.Release()
	c.audio.Release()
}

type codecService struct {
	codecCtx *gmf.CodecCtx
	codec    *gmf.Codec
	stream   *gmf.Stream
}

func NewCodecService(ctx *gmf.FmtCtx, codecName string) *codecService {
	codec, err := gmf.FindEncoder(codecName)
	if err != nil {
		panic(err)
	}
	codecCtx := gmf.NewCodecCtx(codec)

	stream := ctx.NewStream(codec)
	stream.SetCodecCtx(codecCtx)
	return &codecService{
		codecCtx: codecCtx,
		codec:    codec,
		stream:   stream,
	}
}

func (c *codecService) Release() {
	gmf.Release(c.codecCtx)
}
