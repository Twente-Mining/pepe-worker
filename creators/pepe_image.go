package creators

import (
	"cryptopepe.io/cryptopepe-worker/pepe"
	"cryptopepe.io/cryptopepe-svg/builder/look"
	"cryptopepe.io/cryptopepe-svg/builder"
	"gopkg.in/h2non/bimg.v1"
	"bytes"
	"io"
	"cloud.google.com/go/storage"
	"fmt"
	"context"
	"time"
	"strings"
	"io/ioutil"
)

type PepeImageCreator struct {

	svgBuilder *builder.SVGBuilder

	targetBucket *storage.BucketHandle
}

var OutputFormats = []ImgSetting{
	// Thumbnail for special applications (icons / graphs etc.)
	{32, bimg.JPEG},
	// Default size images, for normal use
	{256, bimg.PNG},
	{256, bimg.JPEG},
	// Large resolution for social media, their services will make their own thumbnails
	{2048, bimg.JPEG},
	// SVG is also included, but separate (no image processing required)
}

func NewPepeImageCreator(targetBucket *storage.BucketHandle) *PepeImageCreator {

	creator := new(PepeImageCreator)
	creator.svgBuilder = new(builder.SVGBuilder)
	creator.svgBuilder.Load()

	creator.targetBucket = targetBucket

	return creator

	//worker.bioGenerator = new (bio_gen.BioGenerator)
	//worker.bioGenerator.Load()
}

type ImgSetting struct {
	size int
	format bimg.ImageType
}

func (setting *ImgSetting) GetFilename(pepeId uint64) string {
	return fmt.Sprintf("pepe_%d_%d.%s", pepeId, setting.size, bimg.ImageTypes[setting.format])
}

type StoreImgFn func(filename string, imgBuf []byte, forceOverwrite bool) error


func (creator *PepeImageCreator) Create(pepeId uint64, pepe *pepe.Pepe, look *look.PepeLook, forceOverwrite bool) error {

	// Switch to local storage for debugging
	//var imgStoreFn StoreImgFn = creator.saveImg
	var imgStoreFn StoreImgFn = creator.uploadImg

	// Create the SVG
	buf := new(bytes.Buffer)
	err := creator.svgBuilder.ConvertToSVG(buf, look)
	if err != nil {
		return err
	}

	svgImgFilename := fmt.Sprintf("pepe_%d.svg", pepeId)
	if err := imgStoreFn(svgImgFilename, buf.Bytes(), forceOverwrite); err != nil {
		// Ignore, likely a pre-condition of storage not being met. I.e. image already exists.
		// If writing truely failed, then it will be fixed automatically by the next backfill.
		fmt.Println("Error when writing "+svgImgFilename+" to cloud storage. ", err)
	}

	for _, format := range OutputFormats {
		// Recreate SVG for every output, to avoid quality loss due to image processing
		svgBimg := bimg.NewImage(buf.Bytes())
		output, err := creator.makeImg(svgBimg, format.size, format.format)
		if err != nil {
			return err
		}
		filename := format.GetFilename(pepeId)
		if err := imgStoreFn(filename, output, forceOverwrite); err != nil {
			return err
		}
	}

	return nil
}

func (creator *PepeImageCreator) makeImg(img *bimg.Image, size int, imgType bimg.ImageType) ([]byte, error) {
	processOptions := bimg.Options{
		Width:  size,
		Height: size,
		Type: imgType,
		StripMetadata: true,
		Compression: 9,
		Lossless: false,
	}

	// Convert SVG to PNG
	return img.Process(processOptions)
}


func (creator *PepeImageCreator) saveImg(filename string, imgBuf []byte, forceOverwrite bool) error {
	return ioutil.WriteFile(filename, imgBuf, 0644)
}


func (creator *PepeImageCreator) uploadImg(filename string, imgBuf []byte, forceOverwrite bool) error {
	r := bytes.NewReader(imgBuf)
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second * 3)
	defer cancelFn()
	obj := creator.targetBucket.Object(filename)
	if !forceOverwrite {
		obj = obj.If(storage.Conditions{DoesNotExist: true})
	}
	w := obj.NewWriter(ctx)
	w.ObjectAttrs.CacheControl = "public, max-age=604800" // 7 days cache
	if strings.HasSuffix(filename, ".svg") {
		w.ObjectAttrs.ContentType = "image/svg+xml"
	} else if strings.HasSuffix(filename, ".jpeg") {
		w.ObjectAttrs.ContentType = "image/jpeg"
	} else if strings.HasSuffix(filename, ".png") {
		w.ObjectAttrs.ContentType = "image/png"
	}
	// Everyone can read the image
	w.ObjectAttrs.PredefinedACL = "publicRead"
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		// Ignore precondition fail, that's just because we're not overwriting existing files (given that forceOverwrite is false)
		if forceOverwrite || !strings.Contains(fmt.Sprintf("%s", err), "Precondition Failed") {
			return err
		}
	}
	return nil
}
