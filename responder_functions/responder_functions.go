// Package responder_functions contains all the logic and basic config commands for the responder commands.
package responder_functions

import (
	"bufio"
	"database/sql"
	"fmt"
	owm "github.com/briandowns/openweathermap"
	"github.com/bwmarrin/discordgo"
	"github.com/medzernik/SlovakiaDiscordBotGo/command"
	"github.com/medzernik/SlovakiaDiscordBotGo/config"
	"github.com/medzernik/SlovakiaDiscordBotGo/database"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

const Version string = "0.4.0"

type CommandStatus struct {
	OK      string
	ERR     string
	SYNTAX  string
	WARN    string
	AUTH    string
	AUTOFIX string
}

// CommandStatusBot is a variable to pass to the messageEmbed to make an emoji
var CommandStatusBot CommandStatus = CommandStatus{
	OK:      "",
	ERR:     ":bangbang: ERROR",
	SYNTAX:  ":question: SYNTAX",
	WARN:    ":warning: WARNING",
	AUTH:    ":no_entry: AUTHENTICATION",
	AUTOFIX: ":wrench: AUTOCORRECTING",
}

func Zasielkovna(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) {
	err := command.VerifyArguments(&cmd)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, err.Error())

		return
	}
	s.ChannelMessageSend(m.ChannelID, "OVER 200% <a:medzernikShake:814055147583438848>")

}

// AgeJoined Checks the age of the user on join
func AgeJoined(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) {
	/*
		err := command.VerifyArguments(&cmd, command.RegexArg{Expression: `^<@!(\d+)>$`, CaptureGroup: 1})
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, err.Error())
			return
		}

	*/

	//try to fix command, use only first value
	if len(cmd.Arguments) > 1 {
		command.SendTextEmbed(s, m, CommandStatusBot.AUTOFIX, "usage: **.age @mention** \n Using the first ID instead...", discordgo.EmbedTypeRich)
	}

	//cannot use command
	if len(cmd.Arguments) < 1 {
		command.SendTextEmbed(s, m, CommandStatusBot.SYNTAX, "usage: **.age @mention**", discordgo.EmbedTypeRich)
		return
	}

	userId := command.ParseMentionToString(cmd.Arguments[0])

	//Every time a command is run, get a list of all users. This serves the purpose to then print the name of the corresponding user.
	//TODO: cache it in redis
	membersCached := GetMemberListFromGuild(s, config.Cfg.ServerInfo.GuildIDNumber)

	var userName string

	for i := range membersCached {
		if membersCached[i].User.ID == userId {
			userName = membersCached[i].User.Username
		} else if membersCached[i].User.ID != userId && membersCached[i].User.ID == "" {
			command.SendTextEmbed(s, m, CommandStatusBot.ERR, cmd.Arguments[0]+" : not a number or a mention", discordgo.EmbedTypeRich)
			return
		}
	}

	userTimeRaw, err := SnowflakeTimestamp(userId)
	if err != nil {
		command.SendTextEmbed(s, m, CommandStatusBot.ERR, cmd.Arguments[0]+" : not a number or a mention", discordgo.EmbedTypeRich)
		return
	}

	userTime := time.Now().Sub(userTimeRaw)

	dny := int64(userTime.Hours() / 24)
	hodiny := int64(userTime.Hours()) - dny*24
	minuty := int64(userTime.Minutes()) - int64(userTime.Hours())*60
	sekundy := int64(userTime.Seconds()) - int64(userTime.Minutes())*60

	dnyString := strconv.FormatInt(dny, 10)
	hodinyString := strconv.FormatInt(hodiny, 10)
	minutyString := strconv.FormatInt(minuty, 10)
	sekundyString := strconv.FormatInt(sekundy, 10)

	//send the embed
	command.SendTextEmbed(s, m, CommandStatusBot.OK+userName, command.ParseStringToMentionID(userId)+" je tu s nami už:\n"+
		""+dnyString+" dni\n"+hodinyString+" hodin\n"+
		""+minutyString+" minut\n"+sekundyString+" sekund"+"<:peepoLove:687313976043765810>"+
		"", discordgo.EmbedTypeRich)
}

// Mute Muting function
func Mute(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) {

	//Variable initiation
	var authorisedAdmin bool = false
	var authorisedTrusted bool = false
	authorisedAdmin = command.VerifyAdmin(s, m, &authorisedAdmin, cmd)
	authorisedTrusted = command.VerifyTrusted(s, m, &authorisedTrusted, cmd)

	timeToCheckUsers := 24.0 * -1.0

	//Arguments checking
	if len(cmd.Arguments) < 1 {
		command.SendTextEmbed(s, m, CommandStatusBot.SYNTAX, "usage: **.mute @mention**", discordgo.EmbedTypeRich)
		return
	}

	//Verify, if user has any rights at all
	if authorisedAdmin == false && authorisedTrusted == false {
		command.SendTextEmbed(s, m, CommandStatusBot.AUTH, "Error muting a user - insufficient rights.", discordgo.EmbedTypeRich)
		return
	}

	//Added only after the first check of rights, to prevent spamming of the requests
	membersCached := GetMemberListFromGuild(s, config.Cfg.ServerInfo.GuildIDNumber)
	var MuteUserString []string

	for i := range cmd.Arguments {
		MuteUserString = append(MuteUserString, command.ParseMentionToString(cmd.Arguments[i]))
	}

	//Verify for the admin role before muting.
	if authorisedAdmin == true {
		for i := range membersCached {
			for j := range MuteUserString {
				if membersCached[i].User.ID == MuteUserString[j] {
					//Try to mute
					s.GuildMemberMute(config.Cfg.ServerInfo.GuildIDNumber, MuteUserString[j], true)
					err2 := s.GuildMemberRoleAdd(config.Cfg.ServerInfo.GuildIDNumber, MuteUserString[j], config.Cfg.MuteFunction.MuteRoleID)
					if err2 != nil {
						command.SendTextEmbed(s, m, CommandStatusBot.ERR, "Error muting a user - cannot assign the MuteRole."+
							" "+config.Cfg.MuteFunction.MuteRoleID, discordgo.EmbedTypeRich)
					}
					command.SendTextEmbed(s, m, CommandStatusBot.OK+"MUTED", "Muted user "+command.ParseStringToMentionID(membersCached[i].User.ID)+" (ID: "+
						""+membersCached[i].User.ID+")", discordgo.EmbedTypeRich)
					s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[LOG]** Administrator user "+m.Author.Username+" Muted user: "+
						""+command.ParseStringToMentionID(membersCached[i].User.ID))
				}

			}
		}
	}

	//If not, verify for the role of Trusted to try to mute
	if authorisedTrusted == true && authorisedAdmin == false && config.Cfg.MuteFunction.TrustedMutingEnabled == true {
		for i := range membersCached {
			for j := range MuteUserString {
				userTimeJoin, _ := membersCached[i].JoinedAt.Parse()
				timevar := userTimeJoin.Sub(time.Now()).Hours()
				if membersCached[i].User.ID == MuteUserString[j] && timevar > timeToCheckUsers {
					//Error checking
					s.GuildMemberMute(config.Cfg.ServerInfo.GuildIDNumber, MuteUserString[j], true)

					err2 := s.GuildMemberRoleAdd(config.Cfg.ServerInfo.GuildIDNumber, MuteUserString[j], config.Cfg.MuteFunction.MuteRoleID)
					if err2 != nil {
						command.SendTextEmbed(s, m, CommandStatusBot.ERR, "Error muting a user - cannot assign the MuteRole."+
							" "+config.Cfg.MuteFunction.MuteRoleID, discordgo.EmbedTypeRich)
					}
					command.SendTextEmbed(s, m, CommandStatusBot.OK+"MUTED", "Muted user younger than "+
						""+strconv.FormatInt(int64(timeToCheckUsers*-1.0), 10)+MuteUserString[j], discordgo.EmbedTypeRich)

					s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[LOG]** Trusted user "+command.ParseStringToMentionID(m.Author.ID)+" Muted user: "+
						""+command.ParseStringToMentionID(membersCached[i].User.ID))

					//muting cannot be done if the time limit has been passed
				} else if membersCached[i].User.ID == MuteUserString[j] && timevar < timeToCheckUsers {
					command.SendTextEmbed(s, m, CommandStatusBot.AUTH, "Trusted users cannot mute anyone who has joined more than "+
						""+strconv.FormatInt(int64(timeToCheckUsers*-1.0), 10)+" hours ago.", discordgo.EmbedTypeRich)

				}
			}
		}

	} else if config.Cfg.MuteFunction.TrustedMutingEnabled == false && authorisedTrusted == true && authorisedAdmin == false {
		command.SendTextEmbed(s, m, CommandStatusBot.WARN, "Muting by Trusted users is currently disabled"+
			" "+config.Cfg.MuteFunction.MuteRoleID, discordgo.EmbedTypeRich)
		return
	} else {
		command.SendTextEmbed(s, m, CommandStatusBot.AUTH, "Undefined permissions error"+
			" "+config.Cfg.MuteFunction.MuteRoleID, discordgo.EmbedTypeRich)
		return
	}
	return
}

func KickUser(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) {
	var reason string
	var authorisedAdmin bool = false
	authorisedAdmin = command.VerifyAdmin(s, m, &authorisedAdmin, cmd)

	if len(cmd.Arguments) < 1 {
		command.SendTextEmbed(s, m, CommandStatusBot.SYNTAX, "usage: **.kick @mention reason for kick**", discordgo.EmbedTypeRich)
		return
	}

	if len(cmd.Arguments) > 1 {
		reason = command.JoinArguments(cmd)
	}

	if authorisedAdmin == false {
		command.SendTextEmbed(s, m, CommandStatusBot.AUTH, "Error kicking a user - insufficient rights.", discordgo.EmbedTypeRich)
		return
	}

	membersCached := GetMemberListFromGuild(s, config.Cfg.ServerInfo.GuildIDNumber)

	var KickUserString string = command.ParseMentionToString(cmd.Arguments[0])

	s.ChannelMessageSend(m.ChannelID, "**[PERM]** Permissions check complete.")

	if authorisedAdmin == true {
		for i := range membersCached {
			if membersCached[i].User.ID == KickUserString {
				if len(reason) > 1 {
					//DM the user of his kick + reason
					userNotifChanID, err0 := s.UserChannelCreate(KickUserString)
					if err0 != nil {
						s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[ERR]** Error notifying the user of his kick")
					} else {
						s.ChannelMessageSend(userNotifChanID.ID, "You have been kicked from the server. Reason: "+reason)
					}

					//perform the kick itself
					err := s.GuildMemberDeleteWithReason(config.Cfg.ServerInfo.GuildIDNumber, KickUserString, reason)
					if err != nil {
						command.SendTextEmbed(s, m, CommandStatusBot.ERR, "Error kicking user ID "+membersCached[i].User.ID, discordgo.EmbedTypeRich)
						return
					}

					//log the kick
					command.SendTextEmbed(s, m, CommandStatusBot.OK+"KICKED", "Kicked user "+membersCached[i].User.Username+"for "+
						""+reason, discordgo.EmbedTypeRich)
					s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "User "+KickUserString+" ,was kicked for: "+cmd.Arguments[0]+" .Kicked by "+m.Author.Username)
				} else {
					//DM the user of his kick
					userNotifChanID, err0 := s.UserChannelCreate(KickUserString)
					if err0 != nil {
						s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[ERR]** Error notifying the user of his kick")
					} else {
						s.ChannelMessageSend(userNotifChanID.ID, "You have been kicked from the server.")
					}

					//perform the kick itself
					err := s.GuildMemberDelete(config.Cfg.ServerInfo.GuildIDNumber, KickUserString)
					if err != nil {
						command.SendTextEmbed(s, m, CommandStatusBot.ERR, "Error kicking user ID "+membersCached[i].User.ID, discordgo.EmbedTypeRich)
						return
					}

					//log the kick
					command.SendTextEmbed(s, m, CommandStatusBot.OK+"KICKED", "Kicked user "+membersCached[i].User.Username, discordgo.EmbedTypeRich)
					s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "User "+KickUserString+" "+cmd.Arguments[0]+" Kicked by "+m.Author.Username)
				}
			}

		}
	}
	return
}

func BanUser(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) {
	var reason string
	var daysDelete int = 7
	var authorisedAdmin bool = false
	authorisedAdmin = command.VerifyAdmin(s, m, &authorisedAdmin, cmd)

	if len(cmd.Arguments) < 1 {
		command.SendTextEmbed(s, m, CommandStatusBot.SYNTAX, "usage: **.ban @mention reason for ban**", discordgo.EmbedTypeRich)
		return
	}

	if len(cmd.Arguments) > 1 {
		reason = command.JoinArguments(cmd)
	}

	if authorisedAdmin == false {
		command.SendTextEmbed(s, m, CommandStatusBot.AUTH, "Error banning a user - insufficient rights.", discordgo.EmbedTypeRich)
		return
	}

	membersCached := GetMemberListFromGuild(s, config.Cfg.ServerInfo.GuildIDNumber)

	var BanUserString string = command.ParseMentionToString(cmd.Arguments[0])

	s.ChannelMessageSend(m.ChannelID, "**[PERM]** Permissions check complete.")

	if authorisedAdmin == true {
		for i := range membersCached {
			if membersCached[i].User.ID == BanUserString {
				if len(reason) > 0 {
					userNotifChanID, err0 := s.UserChannelCreate(BanUserString)
					if err0 != nil {
						s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[ERR]** Error notifying the user of his ban")
					} else {
						s.ChannelMessageSend(userNotifChanID.ID, "You have been banned from the server. Reason: "+reason)
					}

					err := s.GuildBanCreateWithReason(config.Cfg.ServerInfo.GuildIDNumber, BanUserString, reason, daysDelete)
					if err != nil {
						command.SendTextEmbed(s, m, CommandStatusBot.ERR, "Error banning user ID "+membersCached[i].User.ID, discordgo.EmbedTypeRich)
						return
					}
					command.SendTextEmbed(s, m, CommandStatusBot.OK+"BANNED", "Banned user "+membersCached[i].User.Username+"for "+
						""+reason, discordgo.EmbedTypeRich)
				} else {
					userNotifChanID, err0 := s.UserChannelCreate(BanUserString)
					if err0 != nil {
						s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[ERR]** Error notifying the user of his ban")
					} else {
						s.ChannelMessageSend(userNotifChanID.ID, "You have been banned from the server.")
					}

					err1 := s.GuildBanCreate(config.Cfg.ServerInfo.GuildIDNumber, BanUserString, daysDelete)
					if err1 != nil {
						command.SendTextEmbed(s, m, CommandStatusBot.ERR, "Error banning user ID "+membersCached[i].User.ID, discordgo.EmbedTypeRich)
						return
					}
					command.SendTextEmbed(s, m, CommandStatusBot.OK+"BANNED", "Banning user "+membersCached[i].User.Username, discordgo.EmbedTypeRich)
					s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "User "+BanUserString+" "+cmd.Arguments[0]+" Banned by "+m.Author.Username)
				}
			}

		}
	}
	return
}

// CheckUsers Checks the age of users
func CheckUsers(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) {
	var timeToCheckUsers float64
	var err error
	if len(cmd.Arguments) > 0 {
		timeToCheckUsers, err = strconv.ParseFloat(cmd.Arguments[0], 64)
		if err != nil {
			command.SendTextEmbed(s, m, CommandStatusBot.AUTOFIX, "usage **.checkusers <numberofhours>**\nUsing default 24 hours...", discordgo.EmbedTypeRich)
			timeToCheckUsers = 24.0 * -1.0
		}
		timeToCheckUsers *= -1.0
	} else {
		timeToCheckUsers = 24.0 * -1.0
	}

	//variable definitions
	var authorisedAdmin bool = false
	authorisedAdmin = command.VerifyAdmin(s, m, &authorisedAdmin, cmd)

	if authorisedAdmin == true {
		membersCached := GetMemberListFromGuild(s, config.Cfg.ServerInfo.GuildIDNumber)
		var mainOutputMsg string
		var IDOutputMsg string

		//iterate over the members_cached array. Maximum limit is 1000.
		for i := range membersCached {
			userTimeJoin, _ := membersCached[i].JoinedAt.Parse()
			var timeVar float64 = userTimeJoin.Sub(time.Now()).Hours()

			if timeVar > timeToCheckUsers {
				mainOutputMsg += "This user is too young (less than " + strconv.FormatFloat(timeToCheckUsers*-1.0, 'f', -1, 64) + "h join age): " + membersCached[i].User.Username + " ,**ID:** " + membersCached[i].User.ID + "\n"
				IDOutputMsg += membersCached[i].User.ID + " "
			}
		}
		//print out the amount of members_cached (max is currently 1000)
		command.SendTextEmbed(s, m, CommandStatusBot.OK+"RECENT USERS", mainOutputMsg+"\n**IDs of the users (copyfriendly):**\n"+IDOutputMsg, discordgo.EmbedTypeRich)
	} else if authorisedAdmin == false {
		command.SendTextEmbed(s, m, CommandStatusBot.AUTH, "You do not have the permission to use this command", discordgo.EmbedTypeRich)
		return
	}
	return

}

// PlanGame Plans a game for a person with a timed reminder
func PlanGame(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) {
	if len(cmd.Arguments) < 3 {
		command.SendTextEmbed(s, m, CommandStatusBot.SYNTAX, "Usage **.plan hh:mm game_name @mention**", discordgo.EmbedTypeRich)
		return
	}
	GamePlanInsert(&cmd, &s, &m)
	return
}

func PlannedGames(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) {
	//open database and then close it (defer)
	sqliteDatabase, _ := sql.Open("sqlite3", "./sqlite-database.db")
	defer func(sqliteDatabase *sql.DB) {
		err := sqliteDatabase.Close()
		if err != nil {
			fmt.Println("error closing the database: ", err)
		}
	}(sqliteDatabase)

	var plannedGames string
	database.DisplayAllGamesPlanned(sqliteDatabase, &plannedGames)

	//send info to channel
	command.SendTextEmbed(s, m, CommandStatusBot.OK+"PLANNED GAMES", plannedGames, discordgo.EmbedTypeRich)
	return
}

// GamePlanInsert Inserts the game into the database
func GamePlanInsert(c *command.Command, s **discordgo.Session, m **discordgo.MessageCreate) {
	//open database and then close it (defer)
	sqliteDatabase, _ := sql.Open("sqlite3", "./sqlite-database.db")
	defer func(sqliteDatabase *sql.DB) {
		err := sqliteDatabase.Close()
		if err != nil {

		}
	}(sqliteDatabase)

	//transform to timestamp
	splitTimeArgument := strings.Split(c.Arguments[0], ":")

	//TODO: Check the capacity if it's sufficient, otherwise the program is panicking every time...
	if cap(splitTimeArgument) < 1 {
		command.SendTextEmbed(*s, *m, CommandStatusBot.ERR, "Error parsing time", discordgo.EmbedTypeRich)

		//(*s).ChannelMessageSend((*m).ChannelID, "**[ERR]** Error parsing time")
		return
	}

	//Put hours into timeHours
	timeHour, err := strconv.Atoi(splitTimeArgument[0])
	if err != nil {
		command.SendTextEmbed(*s, *m, CommandStatusBot.ERR, "Error converting hours", discordgo.EmbedTypeRich)
		//(*s).ChannelMessageSend((*m).ChannelID, "**[ERR]** Error converting hours")
		//fmt.Printf("%s", err)
		return
	}
	//put minutes into timeMinute
	timeMinute, err := strconv.Atoi(splitTimeArgument[1])
	if err != nil {
		command.SendTextEmbed(*s, *m, CommandStatusBot.ERR, "Error converting minutes", discordgo.EmbedTypeRich)
		//(*s).ChannelMessageSend((*m).ChannelID, "**[ERR]** Error converting minutes")
		//fmt.Printf("%s", err)
		return
	}
	//get current date and replace hours and minutes with user variables
	gameTimestamp := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), timeHour, timeMinute, time.Now().Second(), 0, time.Now().Location())
	gameTimestampInt := gameTimestamp.Unix()

	fmt.Println(gameTimestampInt)

	//export to database
	database.InsertGame(sqliteDatabase, gameTimestampInt, c.Arguments[1], c.Arguments[2])

	var plannedgames string
	database.DisplayGamePlanned(sqliteDatabase, &plannedgames)

	command.SendTextEmbed(*s, *m, CommandStatusBot.OK+"PLANNED A GAME", plannedgames, discordgo.EmbedTypeRich)
	return
}

// SnowflakeTimestamp Function to check the user's join date
func SnowflakeTimestamp(ID string) (t time.Time, err error) {
	i, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return
	}
	timestamp := (i >> 22) + 1420070400000
	t = time.Unix(0, timestamp*1000000)
	return
}

// GetMemberListFromGuild Gets the member info
func GetMemberListFromGuild(s *discordgo.Session, guildID string) []*discordgo.Member {
	membersList, err := s.GuildMembers(guildID, "0", 1000)
	if err != nil {
		s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[ERR]** Error getting information about users with the guildID, probably invalid ID?")
	}

	return membersList
}

// CheckRegularSpamAttack Checks the server for spam attacks
func CheckRegularSpamAttack(s *discordgo.Session) {
	//variable definitons
	var membersCached = GetMemberListFromGuild(s, config.Cfg.ServerInfo.GuildIDNumber)
	var tempMsg string
	var spamCounter int64
	var checkInterval time.Duration = 90
	var timeToCheckUsers = 10 * -1.0

	for {
		//iterate over the members_cached array. Maximum limit is 1000.
		for i := range membersCached {
			userTimeJoin, _ := membersCached[i].JoinedAt.Parse()
			timeVar := userTimeJoin.Sub(time.Now()).Minutes()

			if timeVar > timeToCheckUsers {
				tempMsg += "**[ALERT]** RAID PROTECTION ALERT!: User" + membersCached[i].User.Username + "join age: " + strconv.FormatFloat(timeToCheckUsers, 'f', 0, 64) + "\n"
				spamCounter += 1
			}

		}
		if spamCounter > 4 {
			s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[WARN]** Possible RAID ATTACK detected!!! (<@&513275201375698954>) ("+command.ParseStringToMentionID(config.Cfg.RoleAdmin.RoleAdminID+strconv.FormatInt(spamCounter, 10)+" users joined in the last "+strconv.FormatFloat(timeToCheckUsers, 'f', 0, 64)+" hours)"))
		}
		spamCounter = 0
		time.Sleep(checkInterval * time.Second)
	}

}

func Topic(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) {
	fileHandle, err := os.Open("topic_questions.txt")
	if err != nil {
		fmt.Println("error reading the file: ", err)
		command.SendTextEmbed(s, m, CommandStatusBot.ERR, "Error reading the file topic_questions.txt", discordgo.EmbedTypeRich)
		return
	}
	defer func(fileHandle *os.File) {
		err := fileHandle.Close()
		if err != nil {
			fmt.Println("**[ERR]** error closing the file with topics")
		}
	}(fileHandle)

	fileScanner := bufio.NewScanner(fileHandle)

	var splitTopic []string

	for fileScanner.Scan() {
		splitTopic = append(splitTopic, fileScanner.Text())
	}

	//a, b is the length of the topic.
	a := 0
	b := len(splitTopic)

	rand.Seed(time.Now().UnixNano())
	n := a + rand.Intn(b-a+1)

	//this checks the slice length and prevents a panic if for any chance it happened. Just in case.
	if n > len(splitTopic)-1 {
		fmt.Println("[ERR_PARSE] Slice is smaller than allowed\n This error should not have ever happened...")
		return
	}

	command.SendTextEmbed(s, m, CommandStatusBot.OK+"TOPIC", splitTopic[n], discordgo.EmbedTypeRich)
	return
}

func Fox(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "<a:medzernikShake:814055147583438848>")
}

// GetWeather outputs weather information from openWeatherMap
func GetWeather(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) {

	if len(cmd.Arguments) < 1 {
		command.SendTextEmbed(s, m, CommandStatusBot.SYNTAX, "Usage: **.weather City Name**", discordgo.EmbedTypeRich)
		return
	}

	type wData struct {
		name       string
		weather    string
		condition  string
		temp       string
		tempMax    string
		tempMin    string
		tempFeel   string
		pressure   string
		humidity   string
		windSpeed  string
		rainAmount string
		sunrise    string
		sunset     string
	}

	w, err := owm.NewCurrent("C", "en", config.Cfg.ServerInfo.WeatherAPIKey)
	if err != nil {
		fmt.Println("Error processing the request")
		command.SendTextEmbed(s, m, CommandStatusBot.ERR, "Error processing the request", discordgo.EmbedTypeRich)
	}

	var commandString string = command.JoinArguments(cmd)

	err2 := w.CurrentByName(commandString)
	if err2 != nil {
		log.Println(err2)
		command.SendTextEmbed(s, m, CommandStatusBot.ERR, "The city "+commandString+" does not exist", discordgo.EmbedTypeRich)
		return
	}

	var weatherData = wData{
		name:       w.Name,
		weather:    w.Weather[0].Main,
		condition:  w.Weather[0].Description,
		temp:       strconv.FormatFloat(w.Main.Temp, 'f', 1, 64) + " °C",
		tempMax:    strconv.FormatFloat(w.Main.TempMax, 'f', 1, 64) + " °C",
		tempMin:    strconv.FormatFloat(w.Main.TempMin, 'f', 1, 64) + " °C",
		tempFeel:   strconv.FormatFloat(w.Main.FeelsLike, 'f', 1, 64) + " °C",
		pressure:   strconv.FormatFloat(w.Main.Pressure, 'f', 1, 64) + " hPa",
		humidity:   strconv.FormatInt(int64(w.Main.Humidity), 10) + " %",
		windSpeed:  strconv.FormatFloat(w.Wind.Speed, 'f', 1, 64) + " km/h",
		rainAmount: strconv.FormatFloat(w.Rain.OneH*10, 'f', 1, 64) + " %",
		sunrise:    time.Unix(int64(w.Sys.Sunrise), 0).Format(time.Kitchen),
		sunset:     time.Unix(int64(w.Sys.Sunset), 0).Format(time.Kitchen),
	}

	var weatherDataString string = "```\n" +
		"City:\t\t" + weatherData.name + "\n" +
		"Weather:\t" + weatherData.weather + "\n" +
		"Condition:\t" + weatherData.condition + "\n" +
		"Temperature:" + weatherData.temp + "\n" +
		"Max Temp:\t" + weatherData.tempMax + "\n" +
		"Min Temp:\t" + weatherData.tempMin + "\n" +
		"Feel Temp:\t" + weatherData.tempFeel + "\n" +
		"Pressure:\t" + weatherData.pressure + "\n" +
		"Humidity:\t" + weatherData.humidity + "\n" +
		"Wind Speed:\t" + weatherData.windSpeed + "\n" +
		"Rainfall:\t" + weatherData.rainAmount + "\n" +
		"Sunrise:\t" + weatherData.sunrise + "\n" +
		"Sunset:\t" + weatherData.sunset + "\n" +
		"```"

	command.SendTextEmbed(s, m, CommandStatusBot.OK+"WEATHER IN "+strings.ToUpper(w.Name), weatherDataString, discordgo.EmbedTypeRich)
	return

}

// TimedChannelUnlock automatically locks and unlocks a trusted channel
func TimedChannelUnlock(s *discordgo.Session) {
	if config.Cfg.AutoLocker.Enabled == false {
		return
	}

	var checkInterval time.Duration = 60

	fmt.Println("[INIT OK] Channel unlock system module initialized")

	for {
		if time.Now().Weekday() == config.Cfg.AutoLocker.TimeDayUnlock && time.Now().Hour() == config.Cfg.AutoLocker.TimeHourUnlock && time.Now().Minute() == config.Cfg.AutoLocker.TimeMinuteUnlock {
			//Unlock the channel
			//TargetType 0 = roleID, 1 = memberID
			err1 := s.ChannelPermissionSet(config.Cfg.RoleTrusted.ChannelTrustedID, config.Cfg.RoleTrusted.RoleTrustedID1, 0, 2251673408, 0)
			if err1 != nil {
				s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[ERR]** Error changing the permissions for role "+"<@"+config.Cfg.RoleTrusted.RoleTrustedID1+">")
			}
			err2 := s.ChannelPermissionSet(config.Cfg.RoleTrusted.ChannelTrustedID, config.Cfg.RoleTrusted.RoleTrustedID2, 0, 2251673408, 0)
			if err2 != nil {
				s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[ERR]** Error changing the permissions for role "+"<@"+config.Cfg.RoleTrusted.RoleTrustedID2+">")
			}
			err3 := s.ChannelPermissionSet(config.Cfg.RoleTrusted.ChannelTrustedID, config.Cfg.RoleTrusted.RoleTrustedID3, 0, 2251673408, 0)
			if err3 != nil {
				s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[ERR]** Error changing the permissions for role "+"<@"+config.Cfg.RoleTrusted.RoleTrustedID3+">")
			}
			err4 := s.ChannelPermissionSet(config.Cfg.RoleTrusted.ChannelTrustedID, config.Cfg.RoleTrusted.RoleTrustedID4, 0, 2251673408, 0)
			if err4 != nil {
				s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[ERR]** Error changing the permissions for role "+"<@"+config.Cfg.RoleTrusted.RoleTrustedID4+">")
			}

			fmt.Println("[OK] Opened the channel " + config.Cfg.RoleTrusted.ChannelTrustedID)
		} else if time.Now().Weekday() == config.Cfg.AutoLocker.TimeDayLock && time.Now().Hour() == config.Cfg.AutoLocker.TimeHourLock && time.Now().Minute() == config.Cfg.AutoLocker.TimeMinuteLock {
			//Lock the channel
			//TargetType 0 = roleID, 1 = memberID
			err1 := s.ChannelPermissionSet(config.Cfg.RoleTrusted.ChannelTrustedID, config.Cfg.RoleTrusted.RoleTrustedID1, 0, 0, 2251673408)
			if err1 != nil {
				s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[ERR]** Error changing the permissions for role "+"<@"+config.Cfg.RoleTrusted.RoleTrustedID1+">")
			}
			err2 := s.ChannelPermissionSet(config.Cfg.RoleTrusted.ChannelTrustedID, config.Cfg.RoleTrusted.RoleTrustedID2, 0, 0, 2251673408)
			if err2 != nil {
				s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[ERR]** Error changing the permissions for role "+"<@"+config.Cfg.RoleTrusted.RoleTrustedID2+">")
			}
			err3 := s.ChannelPermissionSet(config.Cfg.RoleTrusted.ChannelTrustedID, config.Cfg.RoleTrusted.RoleTrustedID3, 0, 0, 2251673408)
			if err3 != nil {
				s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[ERR]** Error changing the permissions for role "+"<@"+config.Cfg.RoleTrusted.RoleTrustedID3+">")
			}
			err4 := s.ChannelPermissionSet(config.Cfg.RoleTrusted.ChannelTrustedID, config.Cfg.RoleTrusted.RoleTrustedID4, 0, 0, 2251673408)
			if err4 != nil {
				s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[ERR]** Error changing the permissions for role "+"<@"+config.Cfg.RoleTrusted.RoleTrustedID4+">")
			}
			fmt.Println("[OK] Closed the channel " + config.Cfg.RoleTrusted.ChannelTrustedID)
		}

		time.Sleep(checkInterval * time.Second)
	}

}

func PurgeMessages(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) {

	if len(cmd.Arguments) < 1 {
		command.SendTextEmbed(s, m, CommandStatusBot.SYNTAX, "Usage **.purge numberofmessages**", discordgo.EmbedTypeRich)
		return
	}

	var authorisedAdmin bool = false
	authorisedAdmin = command.VerifyAdmin(s, m, &authorisedAdmin, cmd)

	if authorisedAdmin == true {
		var messageArrayToDelete []string

		numMessages, err1 := strconv.ParseInt(cmd.Arguments[0], 10, 64)
		if err1 != nil {
			command.SendTextEmbed(s, m, CommandStatusBot.ERR, "Invalid number provided", discordgo.EmbedTypeRich)
			return
		}
		if numMessages > 99 || numMessages < 1 {
			command.SendTextEmbed(s, m, CommandStatusBot.SYNTAX, "The min-max of the number is 1-100", discordgo.EmbedTypeRich)
			return
		}

		messageArrayComplete, err1 := s.ChannelMessages(m.ChannelID, int(numMessages), m.ID, "", "")
		if err1 != nil {
			command.SendTextEmbed(s, m, CommandStatusBot.ERR, "Cannot get the ID of messages", discordgo.EmbedTypeRich)
			return
		}

		for i := range messageArrayComplete {
			messageArrayToDelete = append(messageArrayToDelete, messageArrayComplete[i].ID)
		}

		err2 := s.ChannelMessagesBulkDelete(m.ChannelID, messageArrayToDelete)
		if err2 != nil {
			command.SendTextEmbed(s, m, CommandStatusBot.ERR, "Error deleting the requested messages...", discordgo.EmbedTypeRich)
			return
		}
		command.SendTextEmbed(s, m, CommandStatusBot.OK+"PURGED", "Purged"+strconv.FormatInt(int64(len(messageArrayToDelete)), 10)+" "+
			"messages", discordgo.EmbedTypeRich)

		s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[LOG]** User "+m.Author.Username+" deleted "+strconv.FormatInt(int64(len(messageArrayToDelete)), 10)+" messages in channel "+"<#"+m.ChannelID+">")

		return
	} else {
		command.SendTextEmbed(s, m, CommandStatusBot.AUTH, "Insufficient permissions.", discordgo.EmbedTypeRich)
		return
	}

}

// Members outputs the number of current members of the server
func Members(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) uint64 {
	if len(cmd.Arguments) > 0 {
		command.SendTextEmbed(s, m, CommandStatusBot.SYNTAX, "Usage: **.count**\n Automatically discarding arguments...", discordgo.EmbedTypeRich)
	}

	memberList := GetMemberListFromGuild(s, config.Cfg.ServerInfo.GuildIDNumber)
	memberListLength := uint64(len(memberList))

	return memberListLength
}

// PruneCount outputs the number of users that could be pruned
func PruneCount(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) uint32 {

	if len(cmd.Arguments) < 1 {
		command.SendTextEmbed(s, m, CommandStatusBot.SYNTAX, "Usage **.prunecount days**", discordgo.EmbedTypeRich)
		return 0
	}

	pruneDaysString := cmd.Arguments[0]
	pruneDaysInt, err1 := strconv.ParseInt(pruneDaysString, 10, 64)
	if err1 != nil {
		command.SendTextEmbed(s, m, CommandStatusBot.ERR, "Error parsing the argument as uint32 days number", discordgo.EmbedTypeRich)
		return 0
	}

	if pruneDaysInt < 7 {
		pruneDaysInt = 0
		command.SendTextEmbed(s, m, CommandStatusBot.WARN, "Command is limited to range 7-30 for safety reasons", discordgo.EmbedTypeRich)
		return 0
	}

	pruneDaysCount, err2 := s.GuildPruneCount(config.Cfg.ServerInfo.GuildIDNumber, uint32(pruneDaysInt))
	if err2 != nil {
		return 0
	}
	return pruneDaysCount

}

// PruneMembers prunes members
func PruneMembers(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) {
	if len(cmd.Arguments) < 100 {
		command.SendTextEmbed(s, m, CommandStatusBot.SYNTAX, "Usage **.prunemembers days**", discordgo.EmbedTypeRich)
		return
	}

	var authorisedAdmin bool = false
	authorisedAdmin = command.VerifyAdmin(s, m, &authorisedAdmin, cmd)

	if authorisedAdmin == true {
		//request prune number amount
		pruneDaysCountInt, err0 := strconv.ParseInt(cmd.Arguments[0], 10, 32)

		if err0 != nil {
			command.SendTextEmbed(s, m, CommandStatusBot.ERR, "Error parsing the argument as uint32 days number", discordgo.EmbedTypeRich)
		}

		var pruneDaysCountUInt = uint32(pruneDaysCountInt)

		if pruneDaysCountInt == 0 {
			command.SendTextEmbed(s, m, CommandStatusBot.SYNTAX, "Cannot prune time of 0 days. Allowed frame is 7-30", discordgo.EmbedTypeRich)
			s.ChannelMessageSend(m.ChannelID, "**[ERR]** Invalid days to prune (0)")
			return
		}

		//prunes the members and assigns the result of the pruned members count to a variable
		prunedMembersCount, err1 := s.GuildPrune(config.Cfg.ServerInfo.GuildIDNumber, pruneDaysCountUInt)
		if err1 != nil {
			command.SendTextEmbed(s, m, CommandStatusBot.ERR, "Error pruning members", discordgo.EmbedTypeRich)
		}

		//log output

		command.SendTextEmbed(s, m, CommandStatusBot.OK+"PRUNED", strconv.FormatInt(int64(prunedMembersCount), 10)+
			" members from the server", discordgo.EmbedTypeRich)
		s.ChannelMessageSend(config.Cfg.ChannelLog.ChannelLogID, "**[LOG]** User "+m.Author.Username+
			" used a prune and kicked "+strconv.FormatInt(int64(prunedMembersCount), 10)+" members")
		return

		//permission output
	} else {
		command.SendTextEmbed(s, m, CommandStatusBot.AUTH, "Insufficient permissions", discordgo.EmbedTypeRich)
		return
	}

}

// MassKick mass kicks user IDs
func MassKick(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) {

}

// MassBan mass bans user IDs
func MassBan(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) {

}

// SetChannelPermission sets a channel permission using an int value
func SetChannelPermission(s *discordgo.Session, cmd command.Command, m *discordgo.MessageCreate) {

}
