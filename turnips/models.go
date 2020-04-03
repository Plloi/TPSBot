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
