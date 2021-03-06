goexif
======

Provides decoding of basic exif and tiff encoded data. Still in alpha - no guarantees.
Suggestions and pull requests are welcome.  Functionality is split into two packages - "exif" and "tiff"
The exif package depends on the tiff package. 
Documentation can be found at http://go.pkgdoc.org/github.com/camlistore/goexif

To install, in a terminal type:

```
go get github.com/camlistore/goexif/exif
```

Or if you just want the tiff package:

```
go get github.com/camlistore/goexif/tiff
```

Example usage:

```go
package main

import (
  "os"
  "log"
  "fmt"

  "github.com/camlistore/goexif/exif"
)

func main() {
  fname := "sample1.jpg"

  f, err := os.Open(fname)
  if err != nil {
    log.Fatal(err)
  }

  x, err := exif.Decode(f)
  f.Close()
  if err != nil {
    log.Fatal(err)
  }

  camModel, _ := x.Get("Model")
  date, _ := x.Get("DateTimeOriginal")
  fmt.Println(camModel.StringVal())
  fmt.Println(date.StringVal())

  focal, _ := x.Get("FocalLength")
  numer, denom := focal.Rat2(0) // retrieve first (only) rat. value
  fmt.Printf("%v/%v", numer, denom)
}
```

<!--golang-->
[![githalytics.com alpha](https://cruel-carlota.pagodabox.com/5e166f74cdb82b999ccd84e3c4dc4348 "githalytics.com")](http://githalytics.com/rwcarlsen/goexif)
