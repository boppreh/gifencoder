package main

import (
    "image"
    "image/color"
    "image/gif"
    "os"
    "bufio"
    "io"
    "compress/lzw"
    "bytes"
    "math"
)

func writeHeader(w *bufio.Writer, image *gif.GIF) {
    w.Write([]uint8("GIF89a"))

    b := image.Image[0].Bounds()
    w.WriteByte(uint8(b.Max.X % 255)) // Paletted width, LSB.
    w.WriteByte(uint8(b.Max.X / 255)) // Paletted width, MSB.
    w.WriteByte(uint8(b.Max.Y % 255)) // Paletted height, LSB.
    w.WriteByte(uint8(b.Max.Y / 255)) // Paletted height, MSB.

    w.WriteByte(uint8(0xF7)) // GCT follows for 256 colors with resolution
                                 // 3 x 8 bits/primary
    w.WriteByte(uint8(0x00)) // Background color.
    w.WriteByte(uint8(0x00)) // Default pixel aspect ratio.

    // Global Color Table.
    palette := image.Image[0].Palette
    for _, c := range palette {
        r, g, b, _ := c.RGBA()
        w.WriteByte(uint8(r))
        w.WriteByte(uint8(g))
        w.WriteByte(uint8(b))
    }

    w.WriteByte(uint8(0x21)) // Application Extension block.
    w.WriteByte(uint8(0xFF)) // Application Extension block (cont).
    w.WriteByte(uint8(0x0B)) // Next 11 bytes are Application Extension.
    w.Write([]uint8("NETSCAPE2.0")) // 8 Character application name.
    w.WriteByte(uint8(0x03)) // 3 more bytes of Application Extension.
    w.WriteByte(uint8(0x01)) // Data sub-block index (always 1).
    w.WriteByte(uint8(0xFF)) // Number of repetitions, LSB.
    w.WriteByte(uint8(0xFF)) // Number of repetitions, MSB.
    w.WriteByte(uint8(0x00)) // End of Application Extension block.
}

func writeFrameHeader(w *bufio.Writer, m *image.Paletted, delay int) {
    w.WriteByte(uint8(0x21)) // Start of Graphic Control Extension.
    w.WriteByte(uint8(0xF9)) // Start of Graphic Control Extension (cont).
    w.WriteByte(uint8(0x04)) // 4 more bytes of GCE.

    w.WriteByte(uint8(0x08)) // There is no transparent pixel.
    w.WriteByte(uint8(delay % 0xFF)) // Animation delay, in centiseconds, LSB.
    w.WriteByte(uint8(delay / 0xFF)) // Animation delay, in centiseconds, MSB.
    w.WriteByte(uint8(0x00)) // Transparent color #, if we were using.
    w.WriteByte(uint8(0x00)) // End of Application Extension data.

    w.WriteByte(uint8(0x2C)) // Start of Paletted Descriptor.

    b := m.Bounds()
    w.WriteByte(uint8(b.Min.X % 255)) // Minimum x (can be > 0), LSB.
    w.WriteByte(uint8(b.Min.X / 255)) // Minimum x (can be > 0), MSB.
    w.WriteByte(uint8(b.Min.Y % 255)) // Minimum y (can be > 0), LSB.
    w.WriteByte(uint8(b.Min.Y / 255)) // Minimum y (can be > 0), MSB.

    w.WriteByte(uint8(b.Max.X % 255)) // Frame width, LSB.
    w.WriteByte(uint8(b.Max.X / 255)) // Frame width, MSB.
    w.WriteByte(uint8(b.Max.Y % 255)) // Frame height, LSB.
    w.WriteByte(uint8(b.Max.Y / 255)) // Frame height, MSB.

    w.WriteByte(uint8(0x00)) // No local color table.
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
            blockSize := math.Min(maxBlockSize, float64(bytesRemaining))
            w.WriteByte(uint8(blockSize))
        }

        b, _ :=  compressedImage.ReadByte()
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
        c := uint8((i / 16) ^ (i % 16))
        p[i] = color.RGBA{c, c * uint8(i), c ^ uint8(i), 0xFF}
    }

    m1 := image.NewPaletted(image.Rect(0, 0, 100, 100), p)
    for x := 0; x < 100; x++ {
        for y := 0; y < 100; y++ {
            m1.SetColorIndex(x, y, uint8(x * y))
        }
    }

    file, _ := os.Create("new_image.gif")

    animation := gif.GIF{[]*image.Paletted{m1}, []int{0}, 0}
    EncodeAll(file, &animation)
}
