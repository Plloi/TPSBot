package turnips

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Plloi/Junior/router"
	"github.com/bwmarrin/discordgo"
	"github.com/sdomino/scribble"
)

func notImplemented(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "Command reserved, but not implemented")
}

// TurnipCommands Holds state form the Turnip Bot
type TurnipCommands struct {
	db     *scribble.Driver
	prices []turnipPrice
	debug  bool
}

// Setup produces a TurnipCommands and assigns it's functions to a router.CommndRouter
func Setup(r *router.CommandRouter) {

	dir := "./turnipdb/"
	db, err := scribble.New(dir, nil)
	if err != nil {
		fmt.Println("Error", err)
	}

	//Load Pricedb
	records, err := db.ReadAll("users")
	if err != nil {
		fmt.Println("Error", err)
	}
	prices := []turnipPrice{}
	for _, f := range records {
		priceRec := turnipPrice{}
		if err := json.Unmarshal([]byte(f), &priceRec); err != nil {
			fmt.Println("Error", err)
		}
		prices = append(prices, priceRec)
	}

	tc := &TurnipCommands{db: db, prices: prices, debug: false}

	r.RegisterCommand("addturnips", "Adds your turnip price to the current list", tc.addturnips)
	r.RegisterCommand("top5", "Show the top 5 turnip prices", tc.top5)
	r.RegisterCommand("settime", "Lets me know what time it is in your game for expiring turnip prices, please send as Wednesday 12 AM", tc.settime)
	r.RegisterCommand("checktime", "I'll tell you what time i think it is in your game", tc.checktime)
	r.RegisterCommand("buyalert", "Set the threshold buy price for the bot to alert you, set to -1 to receive all", tc.buyAlert)
	r.RegisterCommand("sellalert", "Set the threshold sell price for the bot to alert you, set to -1 to receive all", tc.sellAlert)
	// r.RegisterCommand("alertme", "Add me to the alert group", tc.alertme)

}

func (tc *TurnipCommands) top5(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Loop the list looking for best buy and sell prices
	memberList := make(map[string]*discordgo.Member)

	Server, _ := s.Guild(m.GuildID)
	for _, Member := range Server.Members {
		if Member.Nick == "" {
			Member.Nick = Member.User.Username
		}
		memberList[Member.User.ID] = Member
	}

	buying := []turnipPrice{}
	selling := []turnipPrice{}
	sort.SliceStable(tc.prices, func(i, j int) bool {
		return tc.prices[i].Price > tc.prices[j].Price
	})
	for _, price := range tc.prices {
		_, ok := memberList[price.AuthorID]
		if !ok || getOffsetTime(time.Now(), price.TimeOffset).After(price.Time) {
			continue
		}

		if price.Time.Weekday() == 0 {
			selling = append([]turnipPrice{price}, selling...)
		} else {
			buying = append(buying, price)
		}
	}
	msg := ""
	if len(buying) > 0 {
		msg = "Buying:\n"
		for i := 0; i < len(buying) && i < 5; i++ {
			msg += fmt.Sprintf("* %s is buying @ %d\n", memberList[buying[i].AuthorID].Nick, buying[i].Price)
		}
	}

	if len(selling) > 0 {
		msg += "Selling:\n"
		for i := 0; i < len(selling) && i < 5; i++ {
			msg += fmt.Sprintf("* %s is selling @ %d\n", memberList[selling[i].AuthorID].Nick, selling[i].Price)
		}
	}

	if len(msg) == 0 {
		msg = "No prices active at this time"
	}

	s.ChannelMessageSend(m.ChannelID, msg)
}

func (tc *TurnipCommands) addturnips(s *discordgo.Session, m *discordgo.MessageCreate) {
	i, err := strconv.Atoi(m.Content)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "please format message as addturnips 123")
		return
	}

	if i <= 0 {
		s.ChannelMessageSend(m.ChannelID, "Price needs to be greater than 0")
		return
	}

	offset := 0
	userIndex := tc.findUserListingIndex(m.Author.ID)
	if userIndex != -1 {
		offset = tc.prices[userIndex].TimeOffset
	}
	playertime := getOffsetTime(time.Now(), offset)

	//Sunday
	if playertime.Weekday() == 0 && playertime.Hour() >= 5 && playertime.Hour() <= 12 {
		playertime = time.Date(playertime.Year(), playertime.Month(), playertime.Day(), 12, 0, 0, 0, playertime.Location())
		tc.debugLog(fmt.Sprintf("Weekday: %d, Hour: %d, Price: %d, Setting Hour: %d", playertime.Weekday(), playertime.Hour(), i, 12))
		// Weekday morning
	} else if playertime.Weekday() != 0 && playertime.Hour() >= 8 && playertime.Hour() < 12 {
		playertime = time.Date(playertime.Year(), playertime.Month(), playertime.Day(), 12, 0, 0, 0, playertime.Location())
		tc.debugLog(fmt.Sprintf("Weekday: %d, Hour: %d, Price: %d, Setting Hour: %d", playertime.Weekday(), playertime.Hour(), i, 12))
		// Weekday Afternoon
	} else if playertime.Weekday() != 0 && playertime.Hour() >= 12 && playertime.Hour() < 22 {
		playertime = time.Date(playertime.Year(), playertime.Month(), playertime.Day(), 22, 0, 0, 0, playertime.Location())
		tc.debugLog(fmt.Sprintf("Weekday: %d, Hour: %d, Price: %d, Setting Hour: %d", playertime.Weekday(), playertime.Hour(), i, 20))
	} else {
		s.ChannelMessageSend(m.ChannelID, "Aren't you outside of a valid time?")
		return
	}

	var record turnipPrice
	if userIndex == -1 {
		record = turnipPrice{AuthorID: m.Author.ID, Price: i, Time: playertime}
		tc.prices = append(tc.prices, record)
	} else {
		tc.prices[userIndex].Time = playertime
		tc.prices[userIndex].Price = i
		record = tc.prices[userIndex]
	}

	if err := tc.db.Write("users", m.Author.ID, record); err != nil {
		fmt.Println("Error", err)
	}
	s.ChannelMessageSend(m.ChannelID, "Thank you, price added to current market.")

	tc.sendAlert(s, m, record)

}

func (tc *TurnipCommands) settime(s *discordgo.Session, m *discordgo.MessageCreate) {
	now := time.Now()

	regex := regexp.MustCompile(`^((?:sun|mon|tues|wednes|thurs|fri|satur)day) (\d{1,2})\ ?([ap][m])$`)
	matches := regex.FindStringSubmatch(strings.ToLower(m.Content))
	if matches == nil {
		s.ChannelMessageSend(m.ChannelID, "Unable to parse time. please send as 'settime Wednesday 12 PM'")
		return
	}

	hour, _ := strconv.Atoi(matches[2])
	if len(matches) == 4 && matches[3] == "pm" && hour != 12 {
		hour += 12
	} else if len(matches) == 4 && matches[3] == "am" && hour == 12 {
		hour = 0
	}
	offset := hour - now.Hour()

	day, _ := daysOfWeek[matches[1]]

	days := day - now.Weekday()
	fmt.Printf("Adjusting Day by %d Days.\n", days)
	offset = int(days)*24 + offset

	//Find and update, or add new user record
	index := tc.findUserListingIndex(m.Author.ID)
	var record turnipPrice
	if index == -1 {
		record = turnipPrice{AuthorID: m.Author.ID, Price: 0, Time: time.Unix(0, 0), TimeOffset: offset}
		tc.prices = append(tc.prices, record)
	} else {
		tc.prices[index].TimeOffset = offset
		record = tc.prices[index]
	}

	if err := tc.db.Write("users", m.Author.ID, record); err != nil {
		fmt.Println("Error", err)
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s You Game Time Has Been Updated", m.Author.Mention()))
}
func (tc *TurnipCommands) checktime(s *discordgo.Session, m *discordgo.MessageCreate) {
	offset := 0
	userIndex := tc.findUserListingIndex(m.Author.ID)
	if userIndex != -1 {
		offset = tc.prices[userIndex].TimeOffset
	}
	playertime := getOffsetTime(time.Now(), offset)

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s I think your game time is %s", m.Author.Mention(), playertime.Format("Monday 3PM")))

}

func (tc *TurnipCommands) alertme(s *discordgo.Session, m *discordgo.MessageCreate) {
	testtime := time.Now()
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Time now: %s", testtime))
	user := tc.prices[tc.findUserListingIndex(m.Author.ID)]
	testtime = time.Date(testtime.Year(), testtime.Month(), testtime.Day(), testtime.Hour()+user.TimeOffset, testtime.Minute(), testtime.Second(), testtime.Nanosecond(), testtime.Location())
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Time Offset: %s", testtime))
	notImplemented(s, m)
}

func (tc *TurnipCommands) findUserListingIndex(id string) int {
	for i := 0; i < len(tc.prices); i++ {
		if tc.prices[i].AuthorID == id {
			return i
		}
	}
	return -1
}

func getOffsetTime(intime time.Time, offset int) time.Time {
	return time.Date(intime.Year(), intime.Month(), intime.Day(), intime.Hour()+offset, intime.Minute(), intime.Second(), intime.Nanosecond(), intime.Location())
}

func (tc *TurnipCommands) buyAlert(s *discordgo.Session, m *discordgo.MessageCreate) {
	Threshold, err := strconv.Atoi(m.Content)
	if err != nil {
		tc.debugLog(fmt.Sprintf("BuyTreshold encountered an Error:\n Message Content: %s\n Error: %s", m.Content, err))
		s.ChannelMessageSend(m.ChannelID, "Unable to process Alert request")
		return
	}

	userIndex := tc.findUserListingIndex(m.Author.ID)
	var User turnipPrice
	//Existing User
	if userIndex != -1 {
		tc.prices[userIndex].BuyThreshold = Threshold
		User = tc.prices[userIndex]

	} else {
		User.BuyThreshold = Threshold
		User.AuthorID = m.Author.ID
	}

	if err := tc.db.Write("users", m.Author.ID, User); err != nil {
		fmt.Println("Error", err)
	}

	s.ChannelMessageSend(m.ChannelID, "Threshold noted")
}

func (tc *TurnipCommands) sellAlert(s *discordgo.Session, m *discordgo.MessageCreate) {
	Threshold, err := strconv.Atoi(m.Content)
	if err != nil {
		tc.debugLog(fmt.Sprintf("SellTreshold encountered an Error:\n Message Content: %s\n Error: %s", m.Content, err))
		s.ChannelMessageSend(m.ChannelID, "Unable to process Alert request")
		return
	}

	userIndex := tc.findUserListingIndex(m.Author.ID)
	var User turnipPrice
	//Existing User
	if userIndex != -1 {
		tc.prices[userIndex].SellThreshold = Threshold
		User = tc.prices[userIndex]

	} else {
		User.SellThreshold = Threshold
		User.AuthorID = m.Author.ID
	}

	if err := tc.db.Write("users", m.Author.ID, User); err != nil {
		fmt.Println("Error", err)
	}
	s.ChannelMessageSend(m.ChannelID, "Threshold noted")
}

func (tc *TurnipCommands) sendAlert(s *discordgo.Session, m *discordgo.MessageCreate, SourceUser turnipPrice) {
	// Loop the list looking for best buy and sell prices
	memberList := make(map[string]*discordgo.Member)

	Server, _ := s.Guild(m.GuildID)
	for _, Member := range Server.Members {
		if Member.Nick == "" {
			Member.Nick = Member.User.Username
		}
		memberList[Member.User.ID] = Member
	}

	msg := ""

	for _, User := range tc.prices {
		_, ok := memberList[User.AuthorID]
		if !ok || SourceUser.AuthorID == User.AuthorID {
			continue
		}

		//Check Buying
		if (SourceUser.Time.Weekday() != 0) && User.BuyThreshold <= SourceUser.Price && User.BuyThreshold != 0 {
			msg += fmt.Sprintf("%s, ", memberList[User.AuthorID].Mention())
		}

		//Check Selling
		if (SourceUser.Time.Weekday() == 0) && (User.SellThreshold >= SourceUser.Price || User.SellThreshold == -1) {
			msg += fmt.Sprintf("%s, ", memberList[User.AuthorID].Mention())
		}

	}

	if len(msg) > 0 {
		buysell := "buying"
		if SourceUser.Time.Weekday() == 0 {
			buysell = "selling"
		}
		msg = fmt.Sprintf("Hey %s! %s is %s at %d!", msg[0:len(msg)-2], memberList[SourceUser.AuthorID].Nick, buysell, SourceUser.Price)
	}

	s.ChannelMessageSend(m.ChannelID, msg)

}

func (tc *TurnipCommands) debugLog(logentry string) {
	if tc.debug == true {
		fmt.Printf("%s: %s", time.Now(), logentry)
	}
}
