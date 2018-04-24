package form



type C_CreateRepo struct {
	UserID      int64  `binding:"Required"`
	RepoName    string `binding:"Required;AlphaDashDot;MaxSize(100)"`
	DirectoryId int64   `binding:"Required"`
	Private     bool
	Description string `binding:"MaxSize(255)"`
	AutoInit    bool
	Gitignores  string
	License     string
	Readme      string
}
