package handler

import (
	"image"
	"io"
	"log"
	"os"

	e "github.com/julianlk522/fitm/error"
	"github.com/julianlk522/fitm/model"

	"image/gif"
	"image/jpeg"
	"image/png"

	_ "golang.org/x/image/webp"

	"github.com/nfnt/resize"
)

const THUMBNAIL_WIDTH_PX int = 200

// capitalized so it's exported
var (
	Profile_pic_dir, preview_pic_dir string
)

func init() {
	fitm_root_path := os.Getenv("FITM_BACKEND_ROOT")
	if fitm_root_path == "" {
		log.Panic("$FITM_BACKEND_ROOT not set")
	}
	Profile_pic_dir = fitm_root_path + "/db/img/profile"
	preview_pic_dir = fitm_root_path + "/db/img/preview"
}

// First return value is file name of saved image
func SaveUploadedImg(upload *model.ImgUpload) (string, error) {
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

			if !HasAcceptableAspectRatio(img) {
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
		if err = ScaleToThumbnailSize(
			img, 
			file_type, 
			out_file,
		); err != nil {
			return "", err
		}
	} else {
		// reset bytes to beginning
		upload.Bytes.(io.Seeker).Seek(0, 0)
		if _, err = io.Copy(out_file, upload.Bytes); err != nil {
			return "", err
		}
	}

	return file_name, nil
}

func HasAcceptableAspectRatio(img image.Image) bool {
	b := img.Bounds()
	width, height := b.Max.X, b.Max.Y
	ratio := float64(width) / float64(height)

	if ratio > 2.0 || ratio < 0.5 {
		return false
	}

	return true
}

func ScaleToThumbnailSize(img image.Image, file_type string, out_file *os.File) error {
	resized_img := resize.Resize(
		uint(THUMBNAIL_WIDTH_PX), 
		0, 
		img, 
		resize.Lanczos3,
	)

	var err error
	switch file_type {
		case "jpg":
			err = jpeg.Encode(out_file, resized_img, nil)
		case "jpeg":
			err = jpeg.Encode(out_file, resized_img, nil)
		case "png":
			err = png.Encode(out_file, resized_img)
		case "gif":
			err = gif.Encode(out_file, resized_img, nil)
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