// Code generated by "stringer -type=codec"; DO NOT EDIT

package nv

import "fmt"

const _codec_name = "nativeCodecxdrCodec"

var _codec_index = [...]uint8{0, 11, 19}

func (i codec) String() string {
	if i >= codec(len(_codec_index)-1) {
		return fmt.Sprintf("codec(%d)", i)
	}
	return _codec_name[_codec_index[i]:_codec_index[i+1]]
}
