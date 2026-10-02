package pipeline

type Options struct {
	Update         bool
	Keys           bool
	Get            string
	Prefix         string
	FilterRegex    string
	ExcludeRegex   string
	Workers        int
	Force          bool
	KeepRaw        bool
	Convert        bool
	Extract        bool
	Unpack         string
	Root           string
	Version        string
	Encrypt        string
	ExportServer   string
	MasterDownload string
	MasterDecode   string
	MasterEncode   string
	MasterOut      string
	MasterKey      string
	MasterIV       string
	MasterVersion  string
	MasterPrefix   string
	MasterRoot     string
}
