package command

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Command struct {
	Command   string
	Arguments []string
}

type IntegerArg struct {
	LowerLimit int
	UpperLimit int
}

type RegexArg struct {
	Expression   string
	CaptureGroup int
}

var prefix = "."

func ParseCommand(s string) (Command, error) {
	if !strings.HasPrefix(s, prefix) && len(s) < len(prefix)+1 {
		return Command{}, errors.New("parseCommand: Not a command")
	}

	// Remove double white spaces
	space := regexp.MustCompile(`\s+`)
	s = space.ReplaceAllString(s, " ")

	fields := strings.Fields(s)

	cmd := Command{fields[0][len(prefix):], fields[1:]}

	return cmd, nil
}

func IsCommand(c *Command, name string) bool {
	return c.Command == name
}

func VerifyArguments(c *Command, args ...interface{}) error {
	if len(c.Arguments) != len(args) {
		return errors.New(c.Command + ": Incorrect command arguments")
	}

	for i, arg := range args {
		switch t := arg.(type) {
		case int:
			_, err := strconv.ParseInt(c.Arguments[i], 10, 64)
			if err != nil {
				return printArgError(c.Command, c.Arguments[i], "is not a number")
			}

		case string:
			if t != c.Arguments[i] {
				return printArgError(c.Command, c.Arguments[i], "isn't the expected argument "+t)
			}

		case IntegerArg:
			n, err := strconv.Atoi(c.Arguments[i])
			if err != nil || n < t.LowerLimit || n > t.UpperLimit {
				return printArgError(c.Command, c.Arguments[i], "is not a number between"+strconv.Itoa(t.LowerLimit)+
					" and "+strconv.Itoa(t.UpperLimit))
			}

		case RegexArg:
			re, err := regexp.Compile(t.Expression)
			if err != nil {
				return printError(c.Command, "Internal error. Regex for argument["+strconv.Itoa(i)+"] can't be compiled")
			}

			matches := re.FindStringSubmatch(c.Arguments[i])
			if len(matches) == 0 {
				return printArgError(c.Command, c.Arguments[i], "is not a valid argument")
			}

			// Export desired capture-group
			c.Arguments[i] = matches[t.CaptureGroup]

		default:
			return printError(c.Command, "Internal error")

		}
	}

	return nil
}

func printError(command string, cause string) error {
	return fmt.Errorf("%s: %s", command, cause)
}

func printArgError(command string, argument string, cause string) error {
	return printError(command, fmt.Sprintf("Argument \"%s\" %s", argument, cause))
}