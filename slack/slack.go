package slack

type Config struct {
	Url string
}

type Bot struct {
	Config Config
}

type SlackMessage struct {
	ChannelId   string  `form:"channel_id"`
	ChannelName string  `form:"channel_name"`
	ServiceId   string  `form:"service_id"`
	TeamDomain  string  `form:"team_domain"`
	TeamId      string  `form:"team_id"`
	Text        string  `form:"text"`
	Timestamp   float32 `form:"timestamp"`
	Token       string  `form:"token"`
	TriggerWord string  `form:"trigger_word"`
	UserId      string  `form:"user_id"`
	UserName    string  `form:"user_name"`
}
