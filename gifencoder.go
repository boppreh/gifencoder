package main

import (
	"compress/lzw"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"io"
	"os"
)

type encoder struct {
	w io.Writer
	g *gif.GIF
	header [13]byte
	colorTable [3 * 256]byte
	colorTableSize int
	frameHeader [18]byte
	hasTransparent bool
	transparentIndex uint8
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
	b[0] = uint8(p.X)
	b[1] = uint8(p.X >> 8)
	b[2] = uint8(p.Y)
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
	e.header[11] = 0x00 // Background color.
	e.header[12] = 0x00 // Default pixel aspect ratio.
}

func (e *encoder) buildColorTable() {
	// Global Color Table.
	for i, c := range e.g.Image[0].Palette {
		r, g, b, a := c.RGBA()
		e.colorTable[i * 3 + 0] = uint8(r >> 8)
		e.colorTable[i * 3 + 1] = uint8(g >> 8)
		e.colorTable[i * 3 + 2] = uint8(b >> 8)
		if a < 255 {
			e.hasTransparent = true
			e.transparentIndex = uint8(i)
		}
	}
}

func (e *encoder) writeHeader() (err error) {
	e.buildHeader()
	e.buildColorTable()

	_, err = e.w.Write(e.header[:])
	if err != nil {
		return
	}

	_, err = e.w.Write(e.colorTable[:e.colorTableSize * 3])
	if err != nil {
		return
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
    e.frameHeader[4] = e.transparentIndex // Transparent color #, if we are using.
    delay := e.g.Delay[index]
    e.frameHeader[5] = uint8(delay)
    e.frameHeader[6] = uint8(delay >> 8)
    e.frameHeader[7] = uint8(0x00) // End of Application Extension data.

	e.frameHeader[8] = uint8(0x2C) // Start of Paletted Descriptor.
	bounds := e.g.Image[index].Bounds()
	writePoint(e.frameHeader[9:13], bounds.Min)
	writePoint(e.frameHeader[13:17], bounds.Max)
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

	_, err = e.w.Write([]byte{uint8(0x08)}) // Start of LZW with minimum code size 8.
	if err != nil {
		return
	}

	lzww := lzw.NewWriter(&blockWriter{e.w, 0}, lzw.LSB, int(8))
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

func Encode(w io.Writer, m *image.Paletted) error {
	g := gif.GIF{[]*image.Paletted{m}, []int{0}, 0}
	return EncodeAll(w, &g)
}

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

func main() {
	//for _, filename := range []string{"earth", "pattern", "penguin", "newton", "small"} {
    for _, filename := range []string{"small"} {
    	var (err error
    		file *os.File
    		g *gif.GIF
    	)

    	fmt.Println(filename)
        file, _ = os.Open(filename + ".gif")
        g, _ = gif.DecodeAll(file)
        file, _ = os.Create("new_" + filename + ".gif")
        err = EncodeAll(file, g)
        fmt.Println("Encoding error:", err)

        file, _ = os.Open("new_" + filename + ".gif")
        g, err = gif.DecodeAll(file)
        fmt.Println("Encoding error:", err)
    }
}
