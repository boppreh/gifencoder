package main

import (
    "fmt" 
    "image"
    "image/color"
    //"image/gif"
    "os"
    //"bufio"
    "io/ioutil"
    "io"
    "compress/lzw"
    "bytes"
)

func Encode(w io.Writer, m image.Image) {
    file, _ := os.Open("template.gif")
    fileBytes, _ := ioutil.ReadAll(file)

    compressedImageSize := fileBytes[0x320]
    header := fileBytes[:0x320]
    footer := fileBytes[0x321 + uint(compressedImageSize):]

    compressedImageBuffer := bytes.NewBuffer(make([]byte, 0, 255))
    lzww := lzw.NewWriter(compressedImageBuffer, lzw.LSB, int(8))

    b := m.Bounds()
    for y := b.Min.Y; y < b.Max.Y; y++ {
        for x := b.Min.X; x < b.Max.X; x++ {
            c := color.GrayModel.Convert(m.At(x, y)).(color.Gray)
            lzww.Write([]byte{c.Y})
        }
    }
    lzww.Close()

    fmt.Println(compressedImageBuffer.Len())
    
    w.Write(header)
    w.Write([]byte{byte(compressedImageBuffer.Len())})
    compressedImageBuffer.WriteTo(w)
    w.Write(footer)
}

func main() {
    m := image.NewRGBA(image.Rect(0, 0, 52, 52))
    m.Set(5, 5, color.RGBA{0xFF, 0x00, 0x00, 0xFF})
    file, _ := os.Create("new_image.gif")
    Encode(file, m)
}
