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

func Encode(w io.Writer, m image.Image) {
    file, _ := os.Open("template.gif")
    fileBytes, _ := ioutil.ReadAll(file)

    header := fileBytes[:0x320]

    compressedImageBuffer := bytes.NewBuffer(make([]byte, 0, 255))
    lzww := lzw.NewWriter(compressedImageBuffer, lzw.LSB, int(8))

    b := m.Bounds()
    for y := b.Min.Y; y < b.Max.Y; y++ {
        for x := b.Min.X; x < b.Max.X; x++ {
            c := color.GrayModel.Convert(m.At(x, y)).(color.Gray)
            lzww.Write([]byte{c.Y})
            //lzww.Write([]byte{byte(x * y)})
        }
    }
    lzww.Close()

    w.Write(header)
    var bytesSoFar byte = 0
    bytesRemaining := compressedImageBuffer.Len()
    for bytesRemaining > 0 {
        if bytesSoFar == 0 {
            blockSize := math.Min(0xFF, float64(bytesRemaining))
            w.Write([]byte{byte(blockSize)})
        }

        b, _ :=  compressedImageBuffer.ReadByte()
        w.Write([]byte{b})

        bytesSoFar++ // Will overflow and reset the block length count.
        bytesRemaining--
    }
    w.Write([]byte{0, ';'})
}

func main() {
    m := image.NewRGBA(image.Rect(0, 0, 32, 52))
    m.Set(1, 1, color.RGBA{0x00, 0xFF, 0x00, 0xFF})
    file, _ := os.Create("new_image.gif")
    Encode(file, m)
}
