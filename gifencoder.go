package gifencoder

import (
	"compress/lzw"
	"errors"
	"fmt"
	"image"
	"io"
	"image/gif"
)

type encoder struct {
	w                    io.Writer
	g                    *gif.GIF
	header               [13]byte
	colorTable           [3 * 256]byte
	colorTableSize       int
	applicationExtension [19]byte
	frameHeader          [18]byte
	hasTransparent       bool
	transparentIndex     uint8
}

func log2(value int) int {
	// Undefined for value <= 0, but it's used only for the color table size.
	result := -1
	for value > 0 {
		result += 1
		value >>= 1
	}
	return result
}

func writePoint(b []uint8, p image.Point) {
	b[0] = uint8(p.X & 0xFF)
	b[1] = uint8(p.X >> 8)
	b[2] = uint8(p.Y & 0xFF)
	b[3] = uint8(p.Y >> 8)
}

func (e *encoder) buildHeader() {
	e.header[0] = 'G'
	e.header[1] = 'I'
	e.header[2] = 'F'
	e.header[3] = '8'
	e.header[4] = '9'
	e.header[5] = 'a'

	firstImage := e.g.Image[0]

	b := firstImage.Bounds()
	writePoint(e.header[6:10], b.Max)

	e.colorTableSize = len(firstImage.Palette)
	resolution := 8
	// The bits in this in this field mean:
	// 1: The globl color table is present.
	// x \
	// x  |-> Resolution
	// x /
	// 0: The values are not sorted
	// x \
	// x  |-> log2(color table size) - 1
	// x /
	e.header[10] = uint8(0x80 | ((resolution - 1) << 4) | log2(e.colorTableSize) - 1) // Color table information.
	e.header[11] = 0x00                                                               // Background color.
	e.header[12] = 0x00                                                               // Default pixel aspect ratio.
}

func (e *encoder) buildColorTable() {
	// Global Color Table.
	for i, c := range e.g.Image[0].Palette {
		r, g, b, a := c.RGBA()
		e.colorTable[i*3+0] = uint8(r >> 8)
		e.colorTable[i*3+1] = uint8(g >> 8)
		e.colorTable[i*3+2] = uint8(b >> 8)
		if a < 255 {
			e.hasTransparent = true
			e.transparentIndex = uint8(i)
		}
	}
}

func (e *encoder) buildApplicationExtension() {
	e.applicationExtension[0] = 0x21 // Begin application Extension block.
	e.applicationExtension[1] = 0xFF
	e.applicationExtension[2] = 0x0B // Next 11 bytes are Application Extension.
	e.applicationExtension[3] = 'N'  // 8 character application name.
	e.applicationExtension[4] = 'E'
	e.applicationExtension[5] = 'T'
	e.applicationExtension[6] = 'S'
	e.applicationExtension[7] = 'C'
	e.applicationExtension[8] = 'A'
	e.applicationExtension[9] = 'P'
	e.applicationExtension[10] = 'E'
	e.applicationExtension[11] = '2' // 3 character version.
	e.applicationExtension[12] = '.'
	e.applicationExtension[13] = '0'
	e.applicationExtension[14] = 0x03                        // 3 more bytes of Application Extension.
	e.applicationExtension[15] = 0x01                        // Data sub-block index (always 1).
	e.applicationExtension[16] = uint8(e.g.LoopCount & 0xFF) // Number of repetitions.
	e.applicationExtension[17] = uint8(e.g.LoopCount >> 8)
	e.applicationExtension[18] = 0x00 // End of Application Extension block.
}

func (e *encoder) writeHeader() (err error) {
	e.buildHeader()
	e.buildColorTable()

	_, err = e.w.Write(e.header[:])
	if err != nil {
		return
	}

	_, err = e.w.Write(e.colorTable[:e.colorTableSize*3])
	if err != nil {
		return
	}

	if len(e.g.Image) > 1 {
		e.buildApplicationExtension()
		_, err = e.w.Write(e.applicationExtension[:])
		if err != nil {
			return
		}
	}

	return nil
}

func (e *encoder) buildFrameHeader(index int) {
	e.frameHeader[0] = uint8(0x21) // Start of Graphic Control Extension.
	e.frameHeader[1] = uint8(0xF9)
	e.frameHeader[2] = uint8(0x04) // Size of GCE.

	// The bits in this in this field mean:
	// x: Transparent color flag.
	// 0: User input (wait for user input before switching frames).
	// 0 \ Disposal method, don't use previous frame as background.
	// 0 /
	// 0: Reserved
	// 0: Reserved
	// 0: Reserved
	// 0: Reserved
	if e.hasTransparent {
		e.frameHeader[3] = uint8(0x01)
	} else {
		e.frameHeader[3] = uint8(0x00)
	}
	delay := e.g.Delay[index]
	e.frameHeader[4] = uint8(delay)
	e.frameHeader[5] = uint8(delay >> 8)
	e.frameHeader[6] = e.transparentIndex // Transparent color #, if we are using.
	e.frameHeader[7] = uint8(0x00) // End of Application Extension data.

	e.frameHeader[8] = uint8(0x2C) // Start of Paletted Descriptor.
	bounds := e.g.Image[index].Bounds()
	writePoint(e.frameHeader[9:13], bounds.Min)
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y
	writePoint(e.frameHeader[13:17], image.Point{width, height})
	e.frameHeader[17] = uint8(0x00) // No local color table, interlace or sorting.
}

const blockSize = 255

type blockWriter struct {
	w io.Writer
	n int
}

func (bw *blockWriter) Write(p []byte) (n int, err error) {
	bytesWritten := 0
	for len(p) > 0 {
		var blockSize uint8
		if len(p) <= 255 {
			blockSize = uint8(len(p))
		} else {
			blockSize = uint8(255)
		}

		_, err = bw.w.Write([]byte{blockSize})
		if err != nil {
			return bytesWritten, err
		}

		n, err := bw.w.Write(p[:blockSize])
		if err != nil {
			return n, err
		}
		bytesWritten += n + 1

		p = p[blockSize:]
	}

	return bytesWritten, nil
}

func (e *encoder) writeFrame(index int) (err error) {
	e.buildFrameHeader(index)
	_, err = e.w.Write(e.frameHeader[:])
	if err != nil {
		return
	}

	codeSize := log2(e.colorTableSize + 2)
	_, err = e.w.Write([]byte{uint8(codeSize)}) // Start of LZW with minimum code size.
	if err != nil {
		return
	}

	lzww := lzw.NewWriter(&blockWriter{e.w, 0}, lzw.LSB, codeSize)
	_, err = lzww.Write(e.g.Image[index].Pix)
	lzww.Close()
	if err != nil {
		return
	}

	_, err = e.w.Write([]byte{uint8(0x00)}) // End of LZW data.
	if err != nil {
		return
	}

	return nil
}

// Encode takes a single *image.Paletted and encodes it to an io.Writer
func Encode(w io.Writer, m *image.Paletted) error {
	g := gif.GIF{[]*image.Paletted{m}, []int{0}, 0}
	return EncodeAll(w, &g)
}

// EncodeAll encodes a gif to an io.Writer.
func EncodeAll(w io.Writer, g *gif.GIF) (err error) {
	if len(g.Image) == 0 {
		return errors.New("Can't encode zero images.")
	}

	if len(g.Image) != len(g.Delay) {
		return errors.New(fmt.Sprintf("Number of images and delays must be equal (%s x %s)", len(g.Image), len(g.Delay)))
	}

	if g.LoopCount < 0 {
		g.LoopCount = 0
	}

	var e encoder
	e.w = w
	e.g = g

	err = e.writeHeader()
	if err != nil {
		return
	}

	for i, _ := range e.g.Image {
		err = e.writeFrame(i)
		if err != nil {
			return
		}
	}

	_, err = w.Write([]byte{';'})
	if err != nil {
		return
	}

	return nil
}
