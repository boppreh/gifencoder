package main

import (
    //"fmt" 
    "image"
    "image/color"
    //"image/gif"
    "os"
    "bufio"
    "io"
    "compress/lzw"
    "bytes"
    "math"
)

func writeHeader(w *bufio.Writer, m image.Image) {
    w.WriteByte('G')
    w.WriteByte('I')
    w.WriteByte('F')
    w.WriteByte('8')
    w.WriteByte('9')
    w.WriteByte('a')

    b := m.Bounds()
    w.WriteByte(byte(b.Max.X % 255)) // Image width, LSB.
    w.WriteByte(byte(b.Max.X / 255)) // Image width, MSB.
    w.WriteByte(byte(b.Max.Y % 255)) // Image height, LSB.
    w.WriteByte(byte(b.Max.Y / 255)) // Image height, MSB.

    w.WriteByte(byte(0xF7)) // GCT follows for 256 colors with resolution
                                 // 3 x 8 bits/primary
    w.WriteByte(byte(0x00)) // Background color.
    w.WriteByte(byte(0x00)) // Default pixel aspect ratio.

    // Grayscale color table.
    for i := 0; i < 256; i++ {
        w.WriteByte(byte(i)) // Repeat the same color value to get
        w.WriteByte(byte(i)) // grayscale.
        w.WriteByte(byte(i))
    }

    w.WriteByte(byte(0x21)) // Application Extension block.
    w.WriteByte(byte(0xFF)) // Application Extension block (cont).
    w.WriteByte(byte(0x0B)) // Next 11 bytes are Application Extension.
    w.WriteByte('N') // 8 Character application name.
    w.WriteByte('E')
    w.WriteByte('T')
    w.WriteByte('S')
    w.WriteByte('C')
    w.WriteByte('A')
    w.WriteByte('P')
    w.WriteByte('E')
    w.WriteByte('2')
    w.WriteByte('.')
    w.WriteByte('0')
    w.WriteByte(byte(0x03)) // 3 more bytes of Application Extension.
    w.WriteByte(byte(0x01)) // Data sub-block index (always 1).
    w.WriteByte(byte(0xFF)) // Number of repetitions, LSB.
    w.WriteByte(byte(0xFF)) // Number of repetitions, MSB.
    w.WriteByte(byte(0x00)) // End of Application Extension block.
}

func writeFrameHeader(w *bufio.Writer, m image.Image) {
    w.WriteByte(byte(0x21)) // Start of Graphic Control Extension.
    w.WriteByte(byte(0xF9)) // Start of Graphic Control Extension (cont).
    w.WriteByte(byte(0x04)) // 4 more bytes of GCE.

    w.WriteByte(byte(0x08)) // There is no transparent pixel.
    w.WriteByte(byte(0x10)) // Animation delay, in centiseconds, LSB.
    w.WriteByte(byte(0x00)) // Animation delay, in centiseconds, MSB.
    w.WriteByte(byte(0x00)) // Transparent color #, if we were using.
    w.WriteByte(byte(0x00)) // End of Application Extension data.

    w.WriteByte(byte(0x2C)) // Start of Image Descriptor.

    b := m.Bounds()
    w.WriteByte(byte(b.Min.X % 255)) // Minimum x (can be > 0), LSB.
    w.WriteByte(byte(b.Min.X / 255)) // Minimum x (can be > 0), MSB.
    w.WriteByte(byte(b.Min.Y % 255)) // Minimum y (can be > 0), LSB.
    w.WriteByte(byte(b.Min.Y / 255)) // Minimum y (can be > 0), MSB.

    w.WriteByte(byte(b.Max.X % 255)) // Frame width, LSB.
    w.WriteByte(byte(b.Max.X / 255)) // Frame width, MSB.
    w.WriteByte(byte(b.Max.Y % 255)) // Frame height, LSB.
    w.WriteByte(byte(b.Max.Y / 255)) // Frame height, MSB.

    w.WriteByte(byte(0x00)) // No local color table.
}

func compressImage(m image.Image) *bytes.Buffer {
    b := m.Bounds()

    compressedImageBuffer := bytes.NewBuffer(make([]byte, 0, 255))
    lzww := lzw.NewWriter(compressedImageBuffer, lzw.LSB, int(8))

    for y := b.Min.Y; y < b.Max.Y; y++ {
        for x := b.Min.X; x < b.Max.X; x++ {
            c := color.GrayModel.Convert(m.At(x, y)).(color.Gray)
            lzww.Write([]byte{c.Y})
        }
    }
    lzww.Close()

    return compressedImageBuffer
}

func writeFrame(w *bufio.Writer, m image.Image) {
    writeFrameHeader(w, m)

    w.WriteByte(byte(0x08)) // Start of LZW with minimum code size 8.

    compressedImage := compressImage(m)

    const maxBlockSize = 255
    bytesSoFar := 0
    bytesRemaining := compressedImage.Len()
    for bytesRemaining > 0 {
        if bytesSoFar == 0 {
            blockSize := math.Min(maxBlockSize, float64(bytesRemaining))
            w.WriteByte(byte(blockSize))
        }

        b, _ :=  compressedImage.ReadByte()
        w.WriteByte(b)

        bytesSoFar = (bytesSoFar + 1) % maxBlockSize
        bytesRemaining--
    }

    w.WriteByte(byte(0x00)) // End of LZW data.
}

func Encode(w io.Writer, m image.Image) error {
    return EncodeAll(w, []image.Image{m})
}

func EncodeAll(w io.Writer, images []image.Image) error {
    buffer := bufio.NewWriter(w)

    writeHeader(buffer, images[0])
    for _, m := range images {
        writeFrame(buffer, m)
    }
    buffer.WriteByte(';')
    buffer.Flush()

    return nil
}

func main() {
    m1 := image.NewRGBA(image.Rect(0, 0, 100, 100))
    for x := 0; x < 100; x++ {
        for y := 0; y < 100; y++ {
            c := byte(x ^ y)
            m1.Set(x, y, color.RGBA{c, c, c, 0xFF})
        }
    }

    m2 := image.NewRGBA(image.Rect(0, 0, 100, 100))
    for x := 0; x < 100; x++ {
        for y := 0; y < 100; y++ {
            c := byte(x * y)
            m2.Set(x, y, color.RGBA{c, c, c, 0xFF})
        }
    }

    file, _ := os.Create("new_image.gif")
    EncodeAll(file, []image.Image{m1, m2})
}
