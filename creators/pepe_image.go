package creators

import (
	"cryptopepe.io/worker/pepe"
	"cryptopepe.io/cryptopepe-svg/builder/look"
	"cryptopepe.io/cryptopepe-svg/builder"
	"gopkg.in/h2non/bimg.v1"
	"bytes"
	"io"
	"cloud.google.com/go/storage"
	"fmt"
	"context"
	"time"
)

type PepeImageCreator struct {

	svgBuilder *builder.SVGBuilder

	targetBucket *storage.BucketHandle
}

var OutputFormats = []ImgSetting{
	{32, bimg.PNG},
	{32, bimg.JPEG},
	{100, bimg.PNG},
	{100, bimg.JPEG},
	{256, bimg.PNG},
	{512, bimg.JPEG},
	{256, bimg.PNG},
	{512, bimg.JPEG},
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


func (creator *PepeImageCreator) Create(pepeId uint64, pepe *pepe.Pepe, look *look.PepeLook, forceOverwrite bool) error {

	// Create the SVG
	buf := new(bytes.Buffer)
	err := creator.svgBuilder.ConvertToSVG(buf, look)
	if err != nil {
		return err
	}

	svgBimg := bimg.NewImage(buf.Bytes())
	svgImgFilename := fmt.Sprintf("pepe_%d.svg", pepeId)
	if err := creator.uploadImg(svgImgFilename, buf.Bytes(), forceOverwrite); err != nil {
		// Ignore, likely a pre-condition of storage not being met. I.e. image already exists.
		// If writing truely failed, then it will be fixed automatically by the next backfill.
		fmt.Println("Error when writing "+svgImgFilename+" to cloud storage. ", err)
	}

	for _, format := range OutputFormats {
		output, err := creator.makeImg(svgBimg, format.size, format.format)
		if err != nil {
			return err
		}
		filename := format.GetFilename(pepeId)
		if err := creator.uploadImg(filename, output, forceOverwrite); err != nil {
			// Ignore, likely a pre-condition of storage not being met. I.e. image already exists.
			fmt.Println("Error when writing "+filename+" to cloud storage. ", err)
		}
	}

	return nil
}

func (creator *PepeImageCreator) makeImg(img *bimg.Image, size int, imgType bimg.ImageType) ([]byte, error) {
	processOptions := bimg.Options{
		Width:  size,
		Height: size,
		Type: imgType,
	}

	// Convert SVG to PNG
	return img.Process(processOptions)
}

func (creator *PepeImageCreator) uploadImg(filename string, imgBuf []byte, forceOverwrite bool) error {
	r := bytes.NewReader(imgBuf)
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second * 3)
	defer cancelFn()
	wc := creator.targetBucket.Object(filename).If(storage.Conditions{DoesNotExist: !forceOverwrite}).NewWriter(ctx)
	if _, err := io.Copy(wc, r); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return nil
}
