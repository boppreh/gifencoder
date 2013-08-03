package gifencoder

import (
  "image/gif"
	"io/ioutil"
	"os"
	"testing"
)

func TestEncode(t *testing.T) {
	for _, filename := range []string{"cat", "matt"} {
		var (
			err  error
			file *os.File
			g    *gif.GIF
		)

		file, err = os.Open("test/" + filename + ".gif")
    if err != nil {
      t.Errorf("os.Open '%s'", err)
    }

    g, err = gif.DecodeAll(file)
    if err != nil {
      t.Errorf("DecodeAll error '%s'", err)
    }

    file, _ = ioutil.TempFile("", filename)
		err = EncodeAll(file, g)
    if err != nil {
      t.Errorf("EncodeAll error '%s'", err)
    }

    file.Seek(0, os.SEEK_SET) // rewind tmp file
		g, err = gif.DecodeAll(file)
    if err != nil {
      t.Errorf("DecodeAll error '%s'", err)
    }
	}
}

