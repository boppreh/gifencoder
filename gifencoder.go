package main

import (
	"bufio"
	"bytes"
	"compress/lzw"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io"
	"os"
)

var (
	errDelay        = errors.New("gif: number of images and delays don't match")
	errNoImage      = errors.New("gif: no images given (needs at least 1)")
	errNegativeLoop = errors.New("gif: loop count can't be negative (use 0 for infinite)")
)

func log2(value int) int {
    // Undefined for value <= 0, but it's used only for the color table size.
    result := -1
    for value > 0 {
        result += 1
        value >>= 1
    }
    return result
}

func writeLittleEndian(value int, w *bufio.Writer) {
    w.WriteByte(uint8(value & 0xFFFF))
    w.WriteByte(uint8(value >> 8))
}

func writeHeader(w *bufio.Writer, image *gif.GIF) {
	w.Write([]uint8("GIF89a"))

	b := image.Image[0].Bounds()
    writeLittleEndian(b.Max.X, w) // Paletted width.
    writeLittleEndian(b.Max.Y, w) // Paletted height.

	palette := image.Image[0].Palette
	colorTableSize := log2(len(palette)) - 1
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
	w.WriteByte(uint8(0x80 | ((resolution - 1) << 4) | colorTableSize)) // Color table information.
    w.WriteByte(uint8(0x00)) // Background color.
	w.WriteByte(uint8(0x00)) // Default pixel aspect ratio.

	// Global Color Table.
	for _, c := range palette {
		r, g, b, _ := c.RGBA()
		w.WriteByte(uint8(r >> 8))
		w.WriteByte(uint8(g >> 8))
		w.WriteByte(uint8(b >> 8))
	}

	// Add animation info if necessary.
	if len(image.Image) > 1 {
		w.WriteByte(uint8(0x21))              // Application Extension block.
		w.WriteByte(uint8(0xFF))              // Application Extension block (cont).
		w.WriteByte(uint8(0x0B))              // Next 11 bytes are Application Extension.
		w.Write([]uint8("NETSCAPE2.0"))       // 8 Character application name.
		w.WriteByte(uint8(0x03))              // 3 more bytes of Application Extension.
		w.WriteByte(uint8(0x01))              // Data sub-block index (always 1).
        writeLittleEndian(image.LoopCount, w) // Number of repetitions.
		w.WriteByte(uint8(0x00))              // End of Application Extension block.
	}
}

func writeFrameHeader(w *bufio.Writer, m *image.Paletted, delay int) {
    if delay > 0 {
        w.WriteByte(uint8(0x21)) // Start of Graphic Control Extension.
        w.WriteByte(uint8(0xF9)) // Start of Graphic Control Extension (cont).
        w.WriteByte(uint8(0x04)) // 4 more bytes of GCE.

        // The bits in this in this field mean:
        // 0: Transparent color flag
        // 0: User input (wait for user input before switching frames)
        // 1 \ Disposal method, use previous frame as background
        // 0 /
        // 0: Reserved
        // 0: Reserved
        // 0: Reserved
        // 0: Reserved
        w.WriteByte(uint8(0x04)) // There is no transparent pixel.

        writeLittleEndian(delay, w) // Animation delay, in centiseconds.
        w.WriteByte(uint8(0x00))    // Transparent color #, if we were using.
        w.WriteByte(uint8(0x00))    // End of Application Extension data.
    }

	w.WriteByte(uint8(0x2C)) // Start of Paletted Descriptor.

	b := m.Bounds()
    writeLittleEndian(b.Min.X, w) // Minimum x (can be > 0).
    writeLittleEndian(b.Min.Y, w) // Minimum y (can be > 0).
    writeLittleEndian(b.Max.X, w) // Frame width.
    writeLittleEndian(b.Max.Y, w) // Frame height.

	w.WriteByte(uint8(0x00)) // No local color table, interlace or sorting.
}

func compressImage(m *image.Paletted) *bytes.Buffer {
	compressedImageBuffer := bytes.NewBuffer(make([]uint8, 0, 255))
	lzww := lzw.NewWriter(compressedImageBuffer, lzw.LSB, int(8))
	lzww.Write(m.Pix)
	lzww.Close()

	return compressedImageBuffer
}

func writeFrame(w *bufio.Writer, m *image.Paletted, delay int) {
	writeFrameHeader(w, m, delay)

	w.WriteByte(uint8(0x08)) // Start of LZW with minimum code size 8.

	compressedImage := compressImage(m)

	const maxBlockSize = 255
	bytesSoFar := 0
	bytesRemaining := compressedImage.Len()
	for bytesRemaining > 0 {
		if bytesSoFar == 0 {
            var blockSize uint8
            if maxBlockSize < bytesRemaining {
                blockSize = maxBlockSize 
            } else {
                blockSize = uint8(bytesRemaining)
            }
			w.WriteByte(blockSize)
		}

		b, _ := compressedImage.ReadByte()
		w.WriteByte(b)

		bytesSoFar = (bytesSoFar + 1) % maxBlockSize
		bytesRemaining--
	}

	w.WriteByte(uint8(0x00)) // End of LZW data.
}

func Encode(w io.Writer, m *image.Paletted) error {
	animation := gif.GIF{[]*image.Paletted{m}, []int{0}, 0}
	return EncodeAll(w, &animation)
}

func EncodeAll(w io.Writer, animation *gif.GIF) error {
	if len(animation.Image) != len(animation.Delay) {
		return errDelay
	}

	if len(animation.Image) == 0 {
		return errNoImage
	}

	if animation.LoopCount < 0 {
		animation.LoopCount = 0
		//return errNegativeLoop
	}

	buffer := bufio.NewWriter(w)

	writeHeader(buffer, animation)
	for i, _ := range animation.Image {
		image := animation.Image[i]
		delay := animation.Delay[i]
		writeFrame(buffer, image, delay)
	}
	buffer.WriteByte(';')
	buffer.Flush()

	return nil
}

func main() {
	p := make([]color.Color, 256)

	for i := 0; i < 256; i++ {
		//c := uint8((i / 16) ^ (i % 16))
		c := uint8(i)
		p[i] = color.RGBA{c, c, c, 0xFF}
	}

	const nImages = 1
	const imageSize = 1

	images := make([]*image.Paletted, nImages)
	delays := make([]int, nImages)

	for i := 0; i < nImages; i++ {
		m := image.NewPaletted(image.Rect(0, 0, imageSize, imageSize), p)
		for x := 0; x < imageSize; x++ {
			for y := 0; y < imageSize; y++ {
				//m.SetColorIndex(x, y, uint8(x*y/(i+1)))
				m.SetColorIndex(x, y, uint8(x))
			}
		}

		images[i] = m
		delays[i] = 30
	}

    for _, filename := range []string{"earth", "pattern", "penguin", "newton", "small"} {
        file, _ := os.Open(filename + ".gif")
        animation, _ := gif.DecodeAll(file)
        file, err := os.Create("new_" + filename + ".gif")
        EncodeAll(file, animation)

        file, _ = os.Open("new_" + filename + ".gif")
        animation, err = gif.DecodeAll(file)
        fmt.Println(err)
    }
}
