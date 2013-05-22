package main

import (
    "fmt" 
    //"image"
    //"image/gif"
    "os"
    //"bufio"
    "io/ioutil"
    //"io"
    "compress/lzw"
    "bytes"
)

func main() {
    file, _ := os.Open("template.gif")
    fileBytes, _ := ioutil.ReadAll(file)

    imageSize := int(fileBytes[0x320])
    compressedImageBytes := fileBytes[0x321:0x321 + imageSize]

    compressedBuffer := bytes.NewBuffer(compressedImageBytes)
    lzwr := lzw.NewReader(compressedBuffer, lzw.LSB, int(8))
    imageBytes, _ := ioutil.ReadAll(lzwr)

    for i := 0; i < imageSize; i++ {
        compressedImageBytes[i] = 0x00
    }
    fmt.Println(imageBytes)

    compressedBuffer.Reset()
    lzww := lzw.NewWriter(compressedBuffer, lzw.LSB, int(8))
    lzww.Write(imageBytes)

    for i := 0; i < imageSize; i++ {
        fileBytes[0x321 + i] = compressedImageBytes[i]
    }
    //fmt.Println(lzwr)
    ioutil.WriteFile("new_image.gif", fileBytes, 0777)
}
