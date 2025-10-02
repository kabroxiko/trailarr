package internal

type YtdlpFlagsConfig struct {
	Quiet            bool    `yaml:"quiet" json:"quiet"`
	NoProgress       bool    `yaml:"noprogress" json:"noprogress"`
	WriteSubs        bool    `yaml:"writesubs" json:"writesubs"`
	WriteAutoSubs    bool    `yaml:"writeautosubs" json:"writeautosubs"`
	EmbedSubs        bool    `yaml:"embedsubs" json:"embedsubs"`
	RemuxVideo       string  `yaml:"remuxvideo" json:"remuxvideo"`
	SubFormat        string  `yaml:"subformat" json:"subformat"`
	SubLangs         string  `yaml:"sublangs" json:"sublangs"`
	RequestedFormats string  `yaml:"requestedformats" json:"requestedformats"`
	Timeout          float64 `yaml:"timeout" json:"timeout"`
	SleepInterval    float64 `yaml:"sleepInterval" json:"sleepInterval"`
	MaxDownloads     int     `yaml:"maxDownloads" json:"maxDownloads"`
	LimitRate        string  `yaml:"limitRate" json:"limitRate"`
	SleepRequests    float64 `yaml:"sleepRequests" json:"sleepRequests"`
	MaxSleepInterval float64 `yaml:"maxSleepInterval" json:"maxSleepInterval"`
}

func DefaultYtdlpFlagsConfig() YtdlpFlagsConfig {
	return YtdlpFlagsConfig{
		Quiet:            true,
		NoProgress:       true,
		WriteSubs:        true,
		WriteAutoSubs:    true,
		EmbedSubs:        true,
		RemuxVideo:       "mkv",
		SubFormat:        "srt",
		SubLangs:         "es.*",
		RequestedFormats: "best[height<=1080]",
		Timeout:          3.0,
		SleepInterval:    5.0,
		MaxDownloads:     5,
		LimitRate:        "30M",
		SleepRequests:    3.0,
		MaxSleepInterval: 120.0,
	}
}
