package handler

import (
	"image"
	"log"
	"os"

	e "github.com/julianlk522/modeep/error"
	"github.com/julianlk522/modeep/model"

	"image/gif"
	"image/jpeg"
	"image/png"

	_ "golang.org/x/image/webp"

	"github.com/nfnt/resize"
)

// capitalized so it's exported
var (
	Profile_pic_dir, preview_pic_dir string
)

func init() {
	modeep_root_path := os.Getenv("MODEEP_BACKEND_ROOT")
	if modeep_root_path == "" {
		log.Panic("$MODEEP_BACKEND_ROOT not set")
	}
	Profile_pic_dir = modeep_root_path + "/db/img/profile"
	preview_pic_dir = modeep_root_path + "/db/img/preview"
}

func SaveUploadedImgAndGetNewFileName(upload *model.ImgUpload) (string, error) {
	// Verify valid image
	img, file_type, err := image.Decode(upload.Bytes)
	if err != nil {
		if err == image.ErrFormat {
			return "", e.ErrInvalidFileType
		} else {
			return "", err
		}
	}

	var path_prefix string
	switch upload.Purpose {
	case "LinkPreview":
		path_prefix = preview_pic_dir
	case "ProfilePic":
		path_prefix = Profile_pic_dir

		if !hasAcceptableAspectRatio(img) {
			return "", e.ErrInvalidProfilePicAspectRatio
		}
	default:
		return "", e.ErrInvalidImgUploadPurpose
	}

	// Create file
	file_name := upload.UID + "." + file_type
	out_file, err := os.Create(path_prefix + "/" + file_name)
	if err != nil {
		return "", err
	}
	defer out_file.Close()

	// Scale down if needed
	// have not yet figured out how to encode as webp... skip for now
	if img.Bounds().Max.X > THUMBNAIL_WIDTH_PX && file_type != "webp" {
		img = scaleToThumbnailSize(img)
	}

	err = encodeImg(img, file_type, out_file)
	if err != nil {
		return "", err
	}

	return file_name, nil
}

func hasAcceptableAspectRatio(img image.Image) bool {
	b := img.Bounds()
	width, height := b.Max.X, b.Max.Y
	ratio := float64(width) / float64(height)

	if ratio > 2.0 || ratio < 0.5 {
		return false
	}

	return true
}

func scaleToThumbnailSize(img image.Image) image.Image {
	return resize.Resize(
		uint(THUMBNAIL_WIDTH_PX),
		0,
		img,
		resize.Lanczos3,
	)
}

func encodeImg(img image.Image, file_type string, out_file *os.File) error {
	var err error
	switch file_type {
	case "jpg":
		err = jpeg.Encode(out_file, img, nil)
	case "jpeg":
		err = jpeg.Encode(out_file, img, nil)
	case "png":
		err = png.Encode(out_file, img)
	case "gif":
		err = gif.Encode(out_file, img, nil)
	case "webp":
		// there is no Encode function for .webp :(
		// skip and use full sized image
		err = e.ErrCannotEncodeAsWebp
	default:
		log.Printf("unknown file type: %s", file_type)
		err = e.ErrInvalidFileType
	}

	if err != nil {
		return err
	}

	return nil
}
