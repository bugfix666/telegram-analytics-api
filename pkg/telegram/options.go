package telegram

type Options struct {
	APIID       int    // Telegram API ID
	APIHash     string // Telegram API Hash
	SessionFile string // путь к файлу сессии
	BotToken    string // если пусто – пользовательский аккаунт
	Proxy       string // socks5://...
}
