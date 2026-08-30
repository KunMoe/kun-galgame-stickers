package dto

type MultilingualText map[string]string

type Sticker struct {
	Sid       int              `json:"sid"`
	Pid       int              `json:"pid"`
	Game      MultilingualText `json:"game"`
	Loli      MultilingualText `json:"loli"`
	Vndb      int              `json:"vndb"`
	Describe  string           `json:"describe"`
	ImageHash string           `json:"image_hash,omitempty"`
	ImageURL  string           `json:"image_url"`
	ThumbURL  string           `json:"thumb_url"`
}

type Pack struct {
	Sid         int              `json:"sid"`
	OwnerUID    int              `json:"owner_uid"`
	Status      int16            `json:"status"`
	Title       MultilingualText `json:"title"`
	Description MultilingualText `json:"description"`
	PreviewPid  int              `json:"preview_pid"`
	PreviewURL  string           `json:"preview_url"`
	Count       int              `json:"count"`
	PublishedAt *string          `json:"published_at,omitempty"`
}

type PackDetail struct {
	Pack
	Stickers []Sticker `json:"stickers"`
}

type CreatePackRequest struct {
	Title       MultilingualText `json:"title"`
	Description MultilingualText `json:"description"`
}

type PatchPackRequest struct {
	Title       MultilingualText `json:"title"`
	Description MultilingualText `json:"description"`
	PreviewPid  *int             `json:"preview_pid"`
}

type CreateStickerRequest struct {
	ImageHash string           `json:"image_hash"`
	Game      MultilingualText `json:"game"`
	Loli      MultilingualText `json:"loli"`
	Vndb      int              `json:"vndb"`
	Describe  string           `json:"describe"`
}

type PatchStickerRequest struct {
	ImageHash *string          `json:"image_hash"`
	Game      MultilingualText `json:"game"`
	Loli      MultilingualText `json:"loli"`
	Vndb      *int             `json:"vndb"`
	Describe  *string          `json:"describe"`
}

type UploadResult struct {
	Hash        string            `json:"hash"`
	URL         string            `json:"url"`
	VariantURLs map[string]string `json:"variant_urls"`
	Width       int               `json:"width"`
	Height      int               `json:"height"`
}
