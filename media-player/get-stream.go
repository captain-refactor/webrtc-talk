package main

import "github.com/3d0c/gmf"

func getStream(streamType int32, ctx *gmf.FmtCtx) *gmf.Stream {
	for i := 0; i < ctx.StreamsCnt(); i++ {
		stream, err := ctx.GetStream(i)
		if err != nil {
			panic(err)
		}
		if stream.Type() == streamType {
			return stream
		}
	}
	return nil
}

func getVideoStream(ctx *gmf.FmtCtx) *gmf.Stream {
	return getStream(gmf.AVMEDIA_TYPE_VIDEO, ctx)
}

func getAudioStream(ctx *gmf.FmtCtx) *gmf.Stream {
	return getStream(gmf.AVMEDIA_TYPE_AUDIO, ctx)
}
