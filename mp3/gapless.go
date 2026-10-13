package mp3

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
)

// MP3 has no native gapless support. Encoders put a Xing or Info header in
// a silent first frame, and LAME (and ffmpeg's libmp3lame) adds the encoder
// delay and the end padding to it. go-mp3 decodes the header frame as
// silence and plays the delay and the padding, which leaves a gap between
// two gapless tracks. Decode trims them the way ffmpeg does.

// decoderDelay is the delay of a standard MP3 synthesis filterbank, in
// samples. The LAME delay and padding fields leave it out.
const decoderDelay = 529

// syncScan bounds the search for the first frame after the ID3v2 tag.
const syncScan = 64 << 10

// frameProbe is the number of bytes of the first frame that the parser
// reads. It covers the header, the side info, the Xing fields and the LAME tag.
const frameProbe = 256

// gapless holds the samples that are not part of the track.
type gapless struct {
	skip    int // at the start: the header frame, the encoder delay and the decoder delay
	padding int // at the end
}

// readGapless reads the gapless metadata at the start of r and returns a
// reader that starts where r started. A seekable r is seeked back, so go-mp3
// can still seek it. Otherwise the bytes read are replayed.
func readGapless(r io.Reader) (io.Reader, gapless) {
	if s, ok := r.(io.ReadSeeker); ok {
		start, err := s.Seek(0, io.SeekCurrent)
		if err != nil {
			return r, gapless{}
		}
		g := parseGapless(s)
		if _, err := s.Seek(start, io.SeekStart); err != nil {
			return r, gapless{}
		}
		return r, g
	}
	var prefix bytes.Buffer
	g := parseGapless(io.TeeReader(r, &prefix))
	return io.MultiReader(&prefix, r), g
}

// parseGapless skips an ID3v2 tag, finds the first MPEG audio frame and
// parses its gapless metadata. It returns the zero value when there is none.
func parseGapless(r io.Reader) gapless {
	br := bufio.NewReaderSize(r, 4096)
	if hdr, err := br.Peek(10); err == nil && string(hdr[:3]) == "ID3" {
		size := int(hdr[6]&0x7f)<<21 | int(hdr[7]&0x7f)<<14 | int(hdr[8]&0x7f)<<7 | int(hdr[9]&0x7f)
		size += 10
		if hdr[5]&0x10 != 0 { // footer present
			size += 10
		}
		if _, err := br.Discard(size); err != nil {
			return gapless{}
		}
	}
	for i := 0; i < syncScan; i++ {
		hdr, err := br.Peek(4)
		if err != nil {
			return gapless{}
		}
		if frameHeaderValid(hdr) {
			frame, _ := br.Peek(frameProbe)
			return parseFrame(frame)
		}
		if _, err := br.Discard(1); err != nil {
			return gapless{}
		}
	}
	return gapless{}
}

// frameHeaderValid reports whether h starts a Layer III frame header.
func frameHeaderValid(h []byte) bool {
	if len(h) < 4 || h[0] != 0xff || h[1]&0xe0 != 0xe0 {
		return false
	}
	version := (h[1] >> 3) & 3
	layer := (h[1] >> 1) & 3
	bitrate := h[2] >> 4
	rate := (h[2] >> 2) & 3
	return version != 1 && layer == 1 && bitrate != 0 && bitrate != 15 && rate != 3
}

// parseFrame parses the Xing, Info or VBRI header and the LAME tag of the
// first frame. frame starts at the frame header.
func parseFrame(frame []byte) gapless {
	if !frameHeaderValid(frame) {
		return gapless{}
	}
	mpeg1 := (frame[1]>>3)&3 == 3
	mono := frame[3]>>6 == 3
	samplesPerFrame := 576
	sideInfo := 9
	switch {
	case mpeg1 && mono:
		samplesPerFrame, sideInfo = 1152, 17
	case mpeg1:
		samplesPerFrame, sideInfo = 1152, 32
	case !mono:
		sideInfo = 17
	}

	// A VBRI header (Fraunhofer) sits at a fixed offset and has no delay fields.
	if len(frame) >= 40 && string(frame[36:40]) == "VBRI" {
		return gapless{skip: samplesPerFrame}
	}

	off := 4 + sideInfo
	if len(frame) < off+8 {
		return gapless{}
	}
	if tag := string(frame[off : off+4]); tag != "Xing" && tag != "Info" {
		return gapless{}
	}
	g := gapless{skip: samplesPerFrame}
	flags := binary.BigEndian.Uint32(frame[off+4:])
	off += 8
	if flags&1 != 0 { // frame count
		off += 4
	}
	if flags&2 != 0 { // byte count
		off += 4
	}
	if flags&4 != 0 { // seek table
		off += 100
	}
	if flags&8 != 0 { // quality
		off += 4
	}
	if len(frame) < off+24 {
		return g
	}
	switch string(frame[off : off+4]) {
	case "LAME", "Lavf", "Lavc":
	default:
		return g
	}
	// 9 bytes of encoder version, then revision, lowpass, replay gain (8),
	// encoding flags and bitrate. Then 12 bits of delay, 12 of padding.
	b := frame[off+21 : off+24]
	delay := int(b[0])<<4 | int(b[1])>>4
	padding := int(b[1]&0x0f)<<8 | int(b[2])
	g.skip += delay + decoderDelay
	g.padding = max(padding-decoderDelay, 0)
	return g
}
