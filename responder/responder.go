package responder

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/medzernik/SlovakiaDiscordBotGo/command"
)

func RegisterPlugin(s *discordgo.Session) {
	s.AddHandler(messageCreated)
	//s.AddHandler(reactionAdded)
	s.AddHandler(ready)

}

//This is the main logic and command file for now
//TODO: implement a config system
//TODO: implement a command parser ..DONE
//TODO: implement a system of an internal user database

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreated(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	cmd, err := command.ParseCommand(m.Content)

	if err != nil {
		println(err.Error())
		return
	}

	// If the message is "ping" reply with "Pong!"
	if command.IsCommand(&cmd, "Zasielkovna") {
		err := command.VerifyArguments(&cmd)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, err.Error())
			return
		}
		s.ChannelMessageSend(m.ChannelID, "OVER 200% <a:medzernikShake:814055147583438848>")
	}

	// If the message is "pong" reply with "Ping!"
	if command.IsCommand(&cmd, "pong") {
		err := command.VerifyArguments(&cmd)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, err.Error())
			return
		}
		s.ChannelMessageSend(m.ChannelID, "Ping!")
	}

	//a personal reward for our founder of the server that tracks his time on the guild
	if command.IsCommand(&cmd, "age") {
		err := command.VerifyArguments(&cmd, command.RegexArg{`^<@!(\d+)>$`, 1})
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, err.Error())
			return
		}

		user_id := cmd.Arguments[0]
		//Every time a command is run, get a list of all users. This serves the purpose to then print the name of the corresponding user.
		//TODO: Make this list a (ideally cached) variable that at least is shared and not run every time a command is run.
		members, _ := s.GuildMembers("513274646406365184", "0", 1000)
		var user_name string

		for itera, _ := range members {
			if members[itera].User.ID == user_id {
				user_name = members[itera].User.Username
				fmt.Println(user_name)
			}
		}

		fmt.Println(user_id)

		user_time_raw, err := SnowflakeTimestamp(user_id)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Zlé ID slováka")

			return
		}

		if user_id < "0" {
			return
		}

		user_time := time.Now().Sub(user_time_raw)
		user_time_days := user_time.Hours() / 24
		user_time_days = user_time_days / 24

		user_time_days_string := user_time.Hours() / 24
		fmt.Println(user_time_days_string)
		user_time_days_string_pure := strconv.FormatFloat(user_time_days_string, 'f', 0, 64)

		user_time_string := user_time.String()
		user_time_string = strings.ReplaceAll(user_time_string, "h", " Hodín\n ")
		user_time_string = strings.ReplaceAll(user_time_string, "m", " Minút\n ")
		user_time_string = strings.ReplaceAll(user_time_string, "s", " Sekúnd ")

		fmt.Println("log -rayman: ", user_time_string)

		s.ChannelMessageSend(m.ChannelID, "**"+user_name+"**"+" je tu s nami už:\n "+user_time_days_string_pure+" (celkovo dní), rozpis:\n----------\n "+user_time_string+"<:peepoLove:687313976043765810>")

	}

	//right now this command checks for any 1000 users on the guild that have a join time less than 24hours, then prints the names one by one.
	//TODO: check if the users can be >1000
	//TODO: implement a raid protection checker that checks every 1 hour for accounts <2 hours of age and if finds more than 5 -> alert the admins
	//TODO: change the output message to be a single message in a single output to protect from spam. Change the information.
	if command.IsCommand(&cmd, "check-users") {
		err := command.VerifyArguments(&cmd)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, err.Error())
			return
		}
		//variable definitons
		members, _ := s.GuildMembers("513274646406365184", "0", 1000)
		var temp_msg string
		time_to_check_users := (24.0 * -1.0)

		//iterate over the members array. Maximum limit is 1000.
		for itera, _ := range members {
			user_time_join, _ := members[itera].JoinedAt.Parse()
			timevar := user_time_join.Sub(time.Now()).Hours()

			fmt.Println(timevar)

			if timevar > time_to_check_users {
				println("THIS USER IS TOO YOUNG")

				temp_msg += "This user is too young (less than 24h join age): " + members[itera].User.Username + "\n"
			}
		}
		//print out the amount of members (max is currently 1000)
		fmt.Println(len(members))
		s.ChannelMessageSend(m.ChannelID, temp_msg)
	}

}

/*
//this function adds a +1 to a specific emoji reaction to an already added one by a user
//TODO: make it a bit more modular and expand the amount of reactions. Ideally a variable level system
func reactionAdded(s *discordgo.Session, mr *discordgo.MessageReactionAdd) {
	if strings.Contains(strings.ToLower(mr.Emoji.Name), "kekw") {

		s.MessageReactionAdd(mr.ChannelID, mr.MessageID, mr.Emoji.APIName())
	}
	if strings.Contains(strings.ToLower(mr.Emoji.Name), "okayChamp") {
		s.MessageReactionAdd(mr.ChannelID, mr.MessageID, mr.Emoji.APIName())
	}

}
*/
// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func ready(s *discordgo.Session, event *discordgo.Ready) {
	// Set the status.
	s.UpdateGameStatus(0, "Welcome to Slovakia")

}

func SnowflakeTimestamp(ID string) (t time.Time, err error) {
	i, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return
	}
	timestamp := (i >> 22) + 1420070400000
	t = time.Unix(0, timestamp*1000000)
	fmt.Println(t)
	return
}
