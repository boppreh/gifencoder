gifencoder
==========

The Go language has support for a lot of things in its library, including images.
Unfortunately half of the [GIF](http://golang.org/pkg/image/gif/) package is missing: you can only decode files, not
encode them. This repository aims to provide the `Encode` and `EncodeAll` functions
to complete the functionality.

Has support for static images and animations (use `Encode` for static images and
`EncodeAll` for animations), variable delay between frames and infinite looping.

I've tested the output in a few different viewers and they all rendered it
correctly, except for HoneyView. I'm still not sure why, and any help is welcome.


All information on the GIF format was taken from the Wikipedia page on [GIF](https://en.wikipedia.org/wiki/Graphics_Interchange_Format)
