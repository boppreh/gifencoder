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
)

func Encode(w io.Writer, m image.Image) {
    file, _ := os.Open("template.gif")
    fileBytes, _ := ioutil.ReadAll(file)

    imageSize := int(fileBytes[0x320])
    header := fileBytes[:0x320]
    footer := fileBytes[0x320 + imageSize:]

    lzww := lzw.NewWriter(w, lzw.LSB, int(8))
    b := m.Bounds()

    w.Write(header)
    for y := 0; y <= b.Min.Y; y++ {
        for x := 0; x <= b.Min.X; x++ {
            c := color.GrayModel.Convert(m.At(x, y)).(color.Gray)
            lzww.Write([]byte{c.Y})
        }
    }
    w.Write(footer)
}

func main() {
    m := image.NewRGBA(image.Rect(0, 0, 52, 52))
    file, _ := os.Create("new_image.gif")
    Encode(file, m)
}
