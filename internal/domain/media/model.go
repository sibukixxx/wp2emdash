package media

type File struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256,omitempty"`
	MIME   string `json:"mime,omitempty"`
	Ext    string `json:"ext"`
}

type Manifest struct {
	BaseDir    string         `json:"base_dir"`
	TotalFiles int            `json:"total_files"`
	TotalBytes int64          `json:"total_bytes"`
	Extensions map[string]int `json:"extensions"`
	Files      []File         `json:"files"`
}
