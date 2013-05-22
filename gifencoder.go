package main

import (
    //"fmt" 
    "image"
    "image/color"
    //"image/gif"
    "os"
    //"bufio"
    "io/ioutil"
    "io"
    "compress/lzw"
    "bytes"
    "math"
)

func Encode(w io.Writer, m image.Image) error {
    file, _ := os.Open("template.gif")
    fileBytes, _ := ioutil.ReadAll(file)

    b := m.Bounds()
    header := fileBytes[:0x320]
    header[7] = byte(b.Max.X / 255)
    header[6] = byte(b.Max.X % 255)
    header[9] = byte(b.Max.Y / 255)
    header[8] = byte(b.Max.Y % 255)

    header[0x31B] = byte(b.Max.X / 255)
    header[0x31A] = byte(b.Max.X % 255)
    header[0x31D] = byte(b.Max.Y / 255)
    header[0x31C] = byte(b.Max.Y % 255)

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
