package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Plloi/TPSBot/turnips"
	"github.com/Plloi/pdb-cmdr/pkg/router"
	"github.com/bwmarrin/discordgo"
)

// Variables used for command line parameters
var (
	Token  string
	Router *router.CommandRouter
)

func init() {

	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	Router = router.NewCommandRouterWithPrefix("t!")
	Router.RegisterCommand("prefix", "Sets the bot command prefix (Admin Locked)", Router.SetPrefix)

	//Import commands module
	turnips.Setup(Router)

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(Router.HandleCommand)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}
