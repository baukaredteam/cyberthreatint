module phishing-monitor

go 1.21

require (
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/gorilla/websocket v1.5.1
	github.com/joho/godotenv v1.5.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/sirupsen/logrus v1.9.3
	github.com/texttheater/golang-levenshtein v1.0.1
	gorm.io/driver/sqlite v1.5.4
	gorm.io/gorm v1.25.5
)

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.17 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
)
