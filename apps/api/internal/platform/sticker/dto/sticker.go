package dto

type MultilingualText map[string]string

type Sticker struct {
	Sid      int              `json:"sid"`
	Pid      int              `json:"pid"`
	Game     MultilingualText `json:"game"`
	Loli     MultilingualText `json:"loli"`
	Vndb     int              `json:"vndb"`
	Describe string           `json:"describe"`
}

type Pack struct {
	Sid        int `json:"sid"`
	PreviewPid int `json:"preview_pid"`
	Count      int `json:"count"`
}

type PackDetail struct {
	Sid      int       `json:"sid"`
	Stickers []Sticker `json:"stickers"`
}
