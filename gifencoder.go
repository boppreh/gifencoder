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

func writeHeader(w io.Writer, m image.Image) {
    buffer := bufio.NewWriter(w)

    buffer.WriteByte('G')
    buffer.WriteByte('I')
    buffer.WriteByte('F')
    buffer.WriteByte('8')
    buffer.WriteByte('9')
    buffer.WriteByte('a')

    b := m.Bounds()
    buffer.WriteByte(byte(b.Max.X / 255)) // Image width, LSB.
    buffer.WriteByte(byte(b.Max.X % 255)) // Image width, MSB.
    buffer.WriteByte(byte(b.Max.Y / 255)) // Image height, LSB.
    buffer.WriteByte(byte(b.Max.Y % 255)) // Image height, MSB.

    buffer.WriteByte(byte(0xF7)) // GCT follows for 256 colors with resolution
                                 // 3 x 8 bits/primary
    buffer.WriteByte(byte(0x00)) // Background color.
    buffer.WriteByte(byte(0x00)) // Default pixel aspect ratio.

    // Grayscale color table.
    for i := 0; i < 256; i++ {
        buffer.WriteByte(byte(i)) // Repeat the same color value to get
        buffer.WriteByte(byte(i)) // grayscale.
        buffer.WriteByte(byte(i))
    }

    buffer.WriteByte(byte(0x21)) // Application Extension block.
    buffer.WriteByte(byte(0xF9)) // Application Extension block (cont).
    buffer.WriteByte(byte(0x0B)) // Next 11 bytes are Application Extension.
    buffer.WriteByte('N') // 8 Character application name.
    buffer.WriteByte('E')
    buffer.WriteByte('T')
    buffer.WriteByte('S')
    buffer.WriteByte('C')
    buffer.WriteByte('A')
    buffer.WriteByte('P')
    buffer.WriteByte('E')
    buffer.WriteByte('2')
    buffer.WriteByte('.')
    buffer.WriteByte('0')
    buffer.WriteByte(byte(0x03)) // 3 more bytes of Application Extension.
    buffer.WriteByte(byte(0x01)) // Data sub-block index (always 1).
    buffer.WriteByte(byte(0xFF)) // Number of repetitions, LSB.
    buffer.WriteByte(byte(0xFF)) // Number of repetitions, MSB.
    buffer.WriteByte(byte(0x00)) // End of Application Extension block.

    buffer.Flush()
}

func writeFrameHeader(w io.Writer, m image.Image) {
    buffer := bufio.NewWriter(w)

    buffer.WriteByte(byte(0x21)) // Start of Graphic Control Extension.
    buffer.WriteByte(byte(0xF9)) // Start of Graphic Control Extension (cont).
    buffer.WriteByte(byte(0x04)) // 4 more bytes of GCE.

    buffer.WriteByte(byte(0x00)) // There is no transparent pixel.
    buffer.WriteByte(byte(0x10)) // Animation delay, in centiseconds, LSB.
    buffer.WriteByte(byte(0x00)) // Animation delay, in centiseconds, MSB.
    buffer.WriteByte(byte(0x00)) // Transparent color #, if we were using.
    buffer.WriteByte(byte(0x00)) // End of Application Extension data.

    buffer.WriteByte(byte(0x2C)) // Start of Image Descriptor.

    b := m.Bounds()
    buffer.WriteByte(byte(b.Min.X / 255))
    buffer.WriteByte(byte(b.Min.X % 255))
    buffer.WriteByte(byte(b.Min.Y / 255))
    buffer.WriteByte(byte(b.Min.Y % 255))

    buffer.WriteByte(byte(b.Max.X / 255))
    buffer.WriteByte(byte(b.Max.X % 255))
    buffer.WriteByte(byte(b.Max.Y / 255))
    buffer.WriteByte(byte(b.Max.Y % 255))

    buffer.WriteByte(byte(0x00)) // No local color table.

    buffer.WriteByte(byte(0x08)) // Start of LZW with minimum code size 8.

    buffer.Flush()
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

func writeBlocks(w io.Writer, m image.Image) {
    compressedImage := compressImage(m)

    const maxBlockSize = 255
    bytesSoFar := 0
    bytesRemaining := compressedImage.Len()
    for bytesRemaining > 0 {
        if bytesSoFar == 0 {
            blockSize := math.Min(maxBlockSize, float64(bytesRemaining))
            w.Write([]byte{byte(blockSize)})
        }

        b, _ :=  compressedImage.ReadByte()
        writeFrameHeader(w, m)
        w.Write([]byte{b})

        bytesSoFar = (bytesSoFar + 1) % maxBlockSize
        bytesRemaining--
    }
}

func Encode(w io.Writer, m image.Image) error {
    return EncodeAll(w, []image.Image{m})
}

func EncodeAll(w io.Writer, images []image.Image) error {
    writeHeader(w, images[0])
    for _, m := range images {
        writeBlocks(w, m)
    }
    w.Write([]byte{0, ';'})

    return nil
}

func main() {
    m := image.NewRGBA(image.Rect(0, 0, 100, 100))
    for x := 0; x < 100; x++ {
        for y := 0; y < 100; y++ {
            c := byte(x ^ y)
            m.Set(x, y, color.RGBA{c, c, c, 0xFF})
        }
    }
    file, _ := os.Create("new_image.gif")
    Encode(file, m)
}
