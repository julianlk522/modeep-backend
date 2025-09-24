package handler

import "time"

const (
	// Link
	MAX_DAILY_LINKS                                        = 50
	MAX_PREVIEW_IMG_WIDTH_PX                               = 200
	YT_VID_URL_REGEX                                       = `^(https?:\/\/)?(www\.)?(youtube\.com|youtu\.be)\/.+`

	// Tag
	PERCENT_OF_MAX_CAT_SCORE_NEEDED_FOR_ASSIGNMENT float32 = 25

	// Treasure Map
	TMAP_CATS_PAGE_LIMIT int                               = 50
	THUMBNAIL_WIDTH_PX int                                 = 200

	// User
	PW_RESET_TOKEN_VALID_DURATION                          = 10 * time.Minute
)