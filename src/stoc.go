package proxy

import (
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Record struct {
	ID        uint `gorm:"primary_key"`
	MailID    string
	From      string
	CreatedAt time.Time
}

var db *gorm.DB

func Run() {
	var err error
	db, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&Record{})
}

func Get(id string) (email string, ok bool) {
	r := new(Record)
	db.Where("mail_id = ?", id).First(r)
	if r.ID > 0 {
		return r.From, true
	}

	return "", false
}

func Save(id, from string) {
	db.Create(&Record{
		MailID:    id,
		From:      from,
		CreatedAt: time.Now(),
	})
}
