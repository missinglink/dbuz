package main

// https://pkg.go.dev/github.com/godbus/dbus/v5

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/urfave/cli/v2"
)

var conn *dbus.Conn

func main() {
	app := cli.NewApp()
	app.Usage = "dbus cli utils"
	app.EnableBashCompletion = true
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "bus",
			Value: "session",
			Usage: "DBUS bus to use [session/system/custom]",
		},
		&cli.StringFlag{
			Name:  "path",
			Value: "/org/dbuz/default",
			Usage: "DBUS path",
		},
		&cli.StringFlag{
			Name:     "name",
			Required: true,
			Usage:    "DBUS name",
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Value:   false,
		},
	}
	app.Before = func(c *cli.Context) error {
		var err error

		switch c.String("bus") {
		case "session":
			conn, err = dbus.ConnectSessionBus()
		case "system":
			conn, err = dbus.ConnectSystemBus()
		default:
			conn, err = dbus.Dial(c.String("bus"))
		}

		if err != nil {
			return fmt.Errorf("Failed to connect to %s bus. %w", c.String("bus"), err)
		}

		return nil
	}
	app.Commands = []*cli.Command{
		{
			Name:    "subscribe",
			Aliases: []string{"sub"},
			Usage:   "subscribe to a signal",
			Action: func(c *cli.Context) error {
				defer conn.Close()

				if err := conn.AddMatchSignal(signalOptions(c)...); err != nil {
					return fmt.Errorf("Signal match error. %w", err)
				}

				ch := make(chan *dbus.Signal, 1)
				conn.Signal(ch)

				for v := range ch {
					if c.Bool("verbose") {
						fmt.Fprintf(os.Stderr, "%+v\n", v)
					}

					fmt.Printf("%s\n", strings.Join(iterfacesToStrings(v.Body), " "))
				}

				return nil
			},
		},
		{
			Name:    "once",
			Aliases: []string{},
			Usage:   "subscribe to a single signal",
			Action: func(c *cli.Context) error {
				defer conn.Close()

				if err := conn.AddMatchSignal(signalOptions(c)...); err != nil {
					return fmt.Errorf("Signal match error. %w", err)
				}

				ch := make(chan *dbus.Signal, 1)
				conn.Signal(ch)

				v := <-ch
				fmt.Printf("%s", strings.Join(iterfacesToStrings(v.Body), " "))

				return nil
			},
		},
		{
			Name:    "publish",
			Aliases: []string{"pub"},
			Usage:   "publish a signal",
			Action: func(c *cli.Context) error {
				defer conn.Close()

				conn.Emit(
					dbus.ObjectPath(c.String("path")),
					c.String("name"),
					stringsToIterfaces(c.Args().Slice())...,
				)

				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func signalOptions(c *cli.Context) []dbus.MatchOption {
	options := make([]dbus.MatchOption, 0, 10)

	// path, either wildcare or verbatim
	path := c.String("path")
	if len(path) > 0 {
		if strings.HasSuffix(path, "/*") {
			options = append(options, dbus.WithMatchPathNamespace(
				dbus.ObjectPath(strings.TrimSuffix(path, "/*"))),
			)
		} else {
			options = append(options, dbus.WithMatchObjectPath(dbus.ObjectPath(path)))
		}
	}

	name := c.String("name")
	if len(name) > 0 {
		sections := strings.Split(name, ".")

		// last section is the signal 'name'
		options = append(options, dbus.WithMatchMember(sections[len(sections)-1]))

		// previous sections are the 'interface'
		if len(sections) > 1 {
			options = append(options, dbus.WithMatchInterface(
				strings.Join(sections[0:len(sections)-1], ".")),
			)
		}
	}

	return options
}

func stringsToIterfaces(input []string) []interface{} {
	var ret = make([]interface{}, len(input))
	for i, str := range input {
		ret[i] = str
	}
	return ret
}

func iterfacesToStrings(input []interface{}) []string {
	var ret = make([]string, len(input))
	for i, str := range input {
		ret[i] = str.(string)
	}
	return ret
}
