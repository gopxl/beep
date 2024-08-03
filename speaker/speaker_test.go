package speaker

import (
	"fmt"
	"io"
	"testing"

	"github.com/gopxl/beep/v2/internal/testtools"
)

func BenchmarkSampleReader_Read(b *testing.B) {
	// note: must be multiples of bytesPerSample
	bufferSizes := []int{64, 512, 8192, 32768}

	for _, bs := range bufferSizes {
		b.Run(fmt.Sprintf("with buffer size %d", bs), func(b *testing.B) {
			s, _ := testtools.RandomDataStreamer(b.N)
			r := newReaderFromStreamer(s)
			buf := make([]byte, bs)

			b.StartTimer()
			for {
				n, err := r.Read(buf)
				if err == io.EOF {
					break
				}
				if err != nil {
					panic(err)
				}
				if n != len(buf) {
					break
				}
			}
		})
	}
}
