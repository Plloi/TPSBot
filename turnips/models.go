package turnips

import "time"

type turnipPrice struct {
	AuthorID      string
	Price         int
	Time          time.Time
	TimeOffset    int
	BuyThreshold  int
	SellThreshold int
}

var daysOfWeek = map[string]time.Weekday{
	"Sunday":    time.Sunday,
	"Monday":    time.Monday,
	"Tuesday":   time.Tuesday,
	"Wednesday": time.Wednesday,
	"Thursday":  time.Thursday,
	"Friday":    time.Friday,
	"Saturday":  time.Saturday,
}
