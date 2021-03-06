package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
)

var commandRegistry map[string]*Command

type Command struct {
	name    string
	help    string
	handler func(*Connection, ...string)
	mobile  bool
}

var infoCommand = &Command{
	name: "info",
	help: "gives you some info about your current position",
	handler: func(conn *Connection, args ...string) {
		fmt.Fprintf(conn, "current planet: %s\n", conn.System().name)
		fmt.Fprintf(conn, "bombs: %d\n", conn.bombs)
		fmt.Fprintf(conn, "money: %d space duckets\n", conn.money)
	},
}

var nearbyCommand = &Command{
	name: "nearby",
	help: "list objects nearby",
	handler: func(conn *Connection, args ...string) {
		system := conn.System()
		neighbors, err := system.Nearby(25)
		if err != nil {
			log_error("unable to get neighbors: %v", err)
			return
		}
		fmt.Fprintf(conn, "--------------------------------------------------------------------------------\n")
		fmt.Fprintf(conn, "%-4s %-20s %s\n", "id", "name", "travel time")
		fmt.Fprintf(conn, "--------------------------------------------------------------------------------\n")
		for _, neighbor := range neighbors {
			other := index[neighbor.id]
			fmt.Fprintf(conn, "%-4d %-20s %v\n", other.id, other.name, system.TravelTimeTo(other))
		}
		fmt.Fprintf(conn, "--------------------------------------------------------------------------------\n")
	},
}

var helpCommand = &Command{
	name: "help",
	help: "helpful things to help you",
	handler: func(conn *Connection, args ...string) {
		msg := `
Star Dragons is a stupid name, but it's the name that Brian suggested.  It has
nothing to do with Dragons.

Anyway, Star Dragons is a game of cunning text-based, real-time strategy.  You
play as some kind of space-faring entity, faring space in your inspecific
space-faring vessel.  If you want a big one, it's big; if you want a small one,
it's small.  If you want a pink one, it's pink, if you want a black one, it's
black.  And so on, and so forth.  It is the space craft of your dreams.  Or
perhaps you are one of those insect-like alien races and you play as the queen.
Yeah, that's the ticket!  You're the biggest baddest queen bug in space.

In Star Dragons, you issue your spacecraft (which is *not* called a Dragon)
textual commands to control it.  The objective of the game is to be the first
person or alien or bug or magical space ponycorn to eradicate three enemy
species.  Right now that is the only win condition.

All of the systems present in Star Dragons are named and positioned after known
exoplanet systems.  When attempting to communicate from one star system to
another, it takes time for the light of your message to reach the other star
systems.  Star systems that are farther away take longer to communicate with.
        `
		msg = strings.TrimSpace(msg)
		fmt.Fprintln(conn, msg)

		if len(args) == 0 {
			fmt.Fprintln(conn, `use the "commands" command for a list of commands.`)
			fmt.Fprintln(conn, `use "help [command-name]" to get info for a specific command.`)
			return
		}
		for _, cmdName := range args {
			cmd, ok := commandRegistry[cmdName]
			if !ok {
				fmt.Fprintf(conn, "no such command: %v\n", cmdName)
				continue
			}
			fmt.Fprintf(conn, "%v: %v\n", cmdName, cmd.help)
		}
	},
}

var commandsCommand = &Command{
	name: "commands",
	help: "gives you a handy list of commands",
	handler: func(conn *Connection, args ...string) {
		names := make([]string, 0, len(commandRegistry))
		for name, _ := range commandRegistry {
			names = append(names, name)
		}
		sort.Strings(names)
		fmt.Fprintln(conn, "--------------------------------------------------------------------------------")
		for _, name := range names {
			cmd := commandRegistry[name]
			fmt.Fprintf(conn, "%-16s %s\n", name, cmd.help)
		}
		fmt.Fprintln(conn, "--------------------------------------------------------------------------------")
	},
}

var scanCommand = &Command{
	name: "scan",
	help: "super duper scan",
	handler: func(conn *Connection, args ...string) {
		if !conn.CanScan() {
			fmt.Fprintf(conn, "scanners are still recharging.  Can scan again in %v\n", conn.NextScan())
			return
		}
		conn.RecordScan()
		system := conn.System()
		log_info("scan sent from %s", system.name)
		for id, _ := range index {
			if id == system.id {
				continue
			}
			delay := system.LightTimeTo(index[id])
			id2 := id
			After(delay, func() {
				scanSystem(id2, system.id)
			})
		}
	},
}

var broadcastCommand = &Command{
	name: "broadcast",
	help: "broadcast a message for all systems to hear",
	handler: func(conn *Connection, args ...string) {
		msg := strings.Join(args, " ")
		system := conn.System()
		log_info("broadcast sent from %s: %v\n", system.name, msg)
		for id, _ := range index {
			if id == system.id {
				continue
			}
			delay := system.LightTimeTo(index[id])
			id2 := id
			After(delay, func() {
				deliverMessage(id2, system.id, msg)
			})
		}
	},
}

var gotoCommand = &Command{
	name: "goto",
	help: "moves to a different system, specified by either name or ID",
	handler: func(conn *Connection, args ...string) {
		dest_name := strings.Join(args, " ")
		to, ok := nameIndex[dest_name]
		if ok {
			move(conn, to)
			return
		}

		id_n, err := strconv.Atoi(dest_name)
		if err != nil {
			fmt.Fprintf(conn, `hmm, I don't know a system by the name "%s", try something else`, dest_name)
			return
		}

		to, ok = index[id_n]
		if !ok {
			fmt.Fprintf(conn, `oh dear, there doesn't seem to be a system with id %d`, id_n)
			return
		}
		move(conn, to)
	},
}

var mineCommand = &Command{
	name: "mine",
	help: "mines the current system for resources",
	handler: func(conn *Connection, args ...string) {
		conn.StartMining()
		var fn func()
		fn = func() {
			if !conn.IsMining() {
				return
			}
			conn.Payout()
			After(500*time.Millisecond, fn)
		}
		After(500*time.Millisecond, fn)
	},
}

var colonizeCommand = &Command{
	name: "colonize",
	help: "establishes a mining colony on the current system",
	handler: func(conn *Connection, arg ...string) {
		system := conn.System()
		var fn func()
		fn = func() {
			reward := int64(rand.NormFloat64()*5.0 + 100.0*system.miningRate)
			if system.colonizedBy != nil {
				system.colonizedBy.Deposit(reward)
				fmt.Fprintf(system.colonizedBy, "mining colony on %s pays you %d space duckets. total: %d space duckets.\n", system.name, reward, system.colonizedBy.money)
			}
			After(5*time.Second, fn)
		}

		if system.colonizedBy != nil {
			system.colonizedBy = conn
			After(5*time.Second, fn)
			return
		}

		if conn.money > 2000 {
			conn.Withdraw(2000)
			system.colonizedBy = conn
			fmt.Fprintf(conn, "set up a mining colony on %s\n", conn.System().name)
			After(5*time.Second, fn)
		} else {
			fmt.Fprintf(conn, "not enough money!  it costs 2000 duckets to start a mining colony\n")
		}
	},
}

var winCommand = &Command{
	name: "win",
	help: "win the game.",
	handler: func(conn *Connection, args ...string) {
		conn.Win()
	},
}

func move(conn *Connection, to *System) {
	start := conn.System()
	start.Leave(conn)

	delay := start.TravelTimeTo(to)
	fmt.Fprintf(conn, "moving to %s. ETA: %v\n", to.name, delay)
	After(delay, func() {
		to.Arrive(conn)
		fmt.Fprintf(conn, "You have arrived at the %s system after a total travel time of %v.\n", to.name, delay)
	})
}

var bombCommand = &Command{
	name: "bomb",
	help: "bombs a system, with a big space bomb",
	handler: func(conn *Connection, args ...string) {
		if conn.bombs < 1 {
			fmt.Fprintf(conn, "no more bombs left! build more bombs!\n")
			return
		}

		dest_name := strings.Join(args, " ")
		to, ok := nameIndex[dest_name]
		if ok {
			bomb(conn, to)
			return
		}

		id_n, err := strconv.Atoi(dest_name)
		if err != nil {
			fmt.Fprintf(conn, `hmm, I don't know a system by the name "%s", try something else\n`, dest_name)
			return
		}

		to, ok = index[id_n]
		if !ok {
			fmt.Fprintf(conn, `oh dear, there doesn't seem to be a system with id %d\n`, id_n)
			return
		}
		if !conn.CanBomb() {
			fmt.Fprintf(conn, "weapons are still reloading.  Can bomb again in %v\n", conn.NextBomb())
			return
		}
		bomb(conn, to)
	},
}

var mkBombCommand = &Command{
	name: "mkbomb",
	help: "make a bomb.  Costs 500 space duckets",
	handler: func(conn *Connection, args ...string) {
		if conn.money < 500 {
			fmt.Fprintf(conn, "not enough money!  Bombs cost 500 space duckets to build, you only have %d in the bank.\n", conn.money)
			return
		}
		conn.Withdraw(500)
		conn.bombs += 1
		fmt.Fprintf(conn, "built a bomb!\n")
		fmt.Fprintf(conn, "bombs: %d\n", conn.bombs)
		fmt.Fprintf(conn, "money: %d space duckets\n", conn.money)
	},
}

func bomb(conn *Connection, to *System) {
	conn.bombs -= 1
	delay := conn.System().BombTimeTo(to)
	fmt.Fprintf(conn, "sending bomb to %s. ETA: %v\n", to.name, delay)
	After(delay, func() {
		to.Bombed(conn)
	})
}

func isCommand(name string) bool {
	_, ok := commandRegistry[name]
	return ok
}

func runCommand(conn *Connection, name string, args ...string) {
	cmd, ok := commandRegistry[name]
	if !ok {
		fmt.Fprintf(conn, "no such command: %s\n", name)
		return
	}

	if conn.dead {
		fmt.Fprintf(conn, "you're dead.\n")
		return
	}

	if conn.InTransit() && !cmd.mobile {
		fmt.Fprintf(conn, "command %s can not be used while in transit\n", name)
		return
	}
	cmd.handler(conn, args...)
}

func registerCommand(c *Command) {
	commandRegistry[c.name] = c
}

func init() {
	commandRegistry = make(map[string]*Command, 16)
	registerCommand(bombCommand)
	registerCommand(broadcastCommand)
	registerCommand(colonizeCommand)
	registerCommand(commandsCommand)
	registerCommand(gotoCommand)
	registerCommand(helpCommand)
	registerCommand(infoCommand)
	registerCommand(mineCommand)
	registerCommand(nearbyCommand)
	registerCommand(scanCommand)
	registerCommand(mkBombCommand)
	registerCommand(winCommand)
}
