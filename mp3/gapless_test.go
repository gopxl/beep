package mp3

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testFrame builds the start of a first frame. hdr is the 4-byte frame
// header, tag is written at the Xing offset, and lame (when set) adds a LAME
// tag with the given delay and padding after a Xing header with no fields.
func testFrame(hdr [4]byte, sideInfo int, tag string, lame bool, delay, padding int) []byte {
	f := make([]byte, frameProbe)
	copy(f, hdr[:])
	off := 4 + sideInfo
	if tag == "VBRI" {
		off = 36
	}
	copy(f[off:], tag)
	off += 8 // tag and zero flags
	if lame {
		copy(f[off:], "LAME3.100")
		f[off+21] = byte(delay >> 4)
		f[off+22] = byte(delay&0x0f)<<4 | byte(padding>>8)
		f[off+23] = byte(padding)
	}
	return f
}

var (
	mpeg1Stereo = [4]byte{0xff, 0xfb, 0x90, 0x00}
	mpeg1Mono   = [4]byte{0xff, 0xfb, 0x90, 0xc0}
	mpeg2Stereo = [4]byte{0xff, 0xf3, 0x90, 0x00}
)

func TestParseFrame(t *testing.T) {
	tests := []struct {
		name  string
		frame []byte
		want  gapless
	}{
		{"info with lame", testFrame(mpeg1Stereo, 32, "Info", true, 576, 672), gapless{skip: 1152 + 576 + 529, padding: 672 - 529}},
		{"xing mono with lame", testFrame(mpeg1Mono, 17, "Xing", true, 576, 1000), gapless{skip: 1152 + 576 + 529, padding: 1000 - 529}},
		{"mpeg2 with lame", testFrame(mpeg2Stereo, 17, "Info", true, 576, 600), gapless{skip: 576 + 576 + 529, padding: 600 - 529}},
		{"padding below decoder delay", testFrame(mpeg1Stereo, 32, "Info", true, 576, 100), gapless{skip: 1152 + 576 + 529}},
		{"xing without lame", testFrame(mpeg1Stereo, 32, "Xing", false, 0, 0), gapless{skip: 1152}},
		{"vbri", testFrame(mpeg1Stereo, 32, "VBRI", false, 0, 0), gapless{skip: 1152}},
		{"audio frame", testFrame(mpeg1Stereo, 32, "", false, 0, 0), gapless{}},
		{"xing at wrong offset", testFrame(mpeg1Stereo, 17, "Xing", true, 576, 672), gapless{}},
		{"not a frame", []byte("not an mp3 frame"), gapless{}},
		{"truncated", testFrame(mpeg1Stereo, 32, "Info", true, 576, 672)[:40], gapless{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseFrame(tt.frame))
		})
	}
}

func TestParseGapless_SkipsID3AndJunk(t *testing.T) {
	var b bytes.Buffer
	b.Write([]byte{'I', 'D', '3', 4, 0, 0, 0, 0, 0x01, 0x00}) // 128-byte tag
	b.Write(make([]byte, 128))
	b.Write([]byte{0, 0, 0xff, 0x00}) // junk with a false sync byte
	b.Write(testFrame(mpeg1Stereo, 32, "Info", true, 576, 672))
	assert.Equal(t, gapless{skip: 1152 + 576 + 529, padding: 672 - 529}, parseGapless(&b))
}
