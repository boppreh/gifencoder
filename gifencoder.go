package main

import (
    //"fmt" 
    "image"
    "image/color"
    //"image/gif"
    "os"
    //"bufio"
    "io"
    "compress/lzw"
    "bytes"
    "math"
)

func Encode(w io.Writer, m image.Image) error {
    b := m.Bounds()

    header := make([]byte, 0x320)

    header[0] = 'G'
    header[1] = 'I'
    header[2] = 'F'
    header[3] = '8'
    header[4] = '9'
    header[5] = 'a'

    header[7] = byte(b.Max.X / 255)
    header[6] = byte(b.Max.X % 255)
    header[9] = byte(b.Max.Y / 255)
    header[8] = byte(b.Max.Y % 255)

    header[0x0B] = byte(0x00) // Background color.
    header[0x0C] = byte(0x00) // Default pixel aspect ratio.

    // Grayscale color table.
    for i := byte(0); i < 255; i++ {
        header[0x0F + i] = i
        header[0x0E + i] = i
        header[0x0D + i] = i
    }

    header[0x30D] = byte(0x21) // GCE data header.
    header[0x30E] = byte(0xF9) // GCE data header (cont).
    header[0x30F] = byte(0x04) // Next 4 bytes are GCE data.
    header[0x310] = byte(0x01) // There is a transparent pixel.
    header[0x311] = byte(0x00) // Animation delay, LSB.
    header[0x312] = byte(0x00) // Animation delay, MSB.
    header[0x313] = byte(0x10) // And it is color #16 (0x10).
    header[0x314] = byte(0x00) // End of GCE data.

    header[0x315] = byte(0x2C) // Start of Image Descriptor.

    header[0x316] = byte(b.Min.X / 255)
    header[0x315] = byte(b.Min.X % 255)
    header[0x318] = byte(b.Min.Y / 255)
    header[0x317] = byte(b.Min.Y % 255)

    header[0x31B] = byte(b.Max.X / 255)
    header[0x31A] = byte(b.Max.X % 255)
    header[0x31D] = byte(b.Max.Y / 255)
    header[0x31C] = byte(b.Max.Y % 255)

    header[0x31E] = byte(0x00) // No local color table.

    header[0x31F] = byte(0x08) // Start of LZW with minimum code size 8.

    compressedImageBuffer := bytes.NewBuffer(make([]byte, 0, 255))
    lzww := lzw.NewWriter(compressedImageBuffer, lzw.LSB, int(8))

    for y := b.Min.Y; y < b.Max.Y; y++ {
        for x := b.Min.X; x < b.Max.X; x++ {
            c := color.GrayModel.Convert(m.At(x, y)).(color.Gray)
            lzww.Write([]byte{c.Y})
            //lzww.Write([]byte{byte(x ^ y)})
            //lzww.Write([]byte{byte(0x00)})
        }
    }
    lzww.Close()

    w.Write(header)

    const maxBlockSize = 255
    bytesSoFar := 0
    bytesRemaining := compressedImageBuffer.Len()
    for bytesRemaining > 0 {
        if bytesSoFar == 0 {
            blockSize := math.Min(maxBlockSize, float64(bytesRemaining))
            w.Write([]byte{byte(blockSize)})
        }

        b, _ :=  compressedImageBuffer.ReadByte()
        w.Write([]byte{b})

        bytesSoFar = (bytesSoFar + 1) % maxBlockSize
        bytesRemaining--
    }

    w.Write([]byte{0, ';'})

    return nil
}

func main() {
    m := image.NewRGBA(image.Rect(0, 0, 100, 100))
    m.Set(1, 1, color.RGBA{0x00, 0xFF, 0x00, 0xFF})
    file, _ := os.Create("new_image.gif")
    Encode(file, m)
}
