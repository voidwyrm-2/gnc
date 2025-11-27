package main

import (
	"bufio"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/term"
)

//go:embed version.txt
var version string

func _main() error {
	version = strings.TrimSpace(version)

	fVersion := flag.Bool("v", false, "Show the current GNC version.")
	fUser := flag.String("u", "", "Sets the username.")

	flag.Parse()

	if *fVersion {
		fmt.Println("Golang Nanochat Client (GNC), version", version)
		return nil
	}

	oldState, err := term.MakeRaw(0)
	if err != nil {
		return err
	}

	defer term.Restore(0, oldState)

	tm := term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}, "> ")

	var (
		addr string
		conn net.Conn
	)

	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	tprint := func(a ...any) {
		fmt.Fprintln(tm, a...)
	}

	tprintf := func(frm string, a ...any) {
		fmt.Fprintf(tm, frm, a...)
	}

loop:
	for {
		l, err := tm.ReadLine()
		if err != nil {
			return err
		} else if len(l) == 0 {
			continue
		}

		cmd := strings.TrimSpace(l)
		txt := ""

		for i, c := range cmd {
			if unicode.IsSpace(rune(c)) {
				txt = strings.TrimSpace(cmd[i:])
				cmd = cmd[:i]
				break
			}
		}

		args := strings.Fields(txt)
		_ = args

		switch cmd {
		case "exit":
			if conn != nil {
				fmt.Fprintln(conn, "QUIT")
			}

			break loop
		case "help":
			tprintf(`Golang Nanochat Client (GNC), version %s

Commands:
 exit - exits the client.
 help - shows this message.
 conn [host] [port] - connects to the specified host and port.
 send <msg...> - sends a message.
 hist - lists the previously sent messages.
 last <n> - show the specified number of previously sent messages.
 poll <n> - show the amount of messages after the specified index.
 skip <n> - gets the earliest messages after the specified index.
`, version)
		case "conn":
			{
				if len(args) != 1 && len(args) != 2 {
					tprint("'conn' expects one or two arguments")
					continue
				}

				if conn != nil {
					tprintf("Disconnecting from '%s'\n", addr)
					fmt.Fprintln(conn, "QUIT")
					conn.Close()
				}

				if len(args) == 1 {
					addr = fmt.Sprintf("%s:44322", args[0])
				} else {
					addr = fmt.Sprintf("%s:%s", args[0], args[1])
				}

				tprintf("Opening a connection to '%s'\n", addr)

				conn, err = net.Dial("tcp", addr)
				if err != nil {
					tprint(err.Error())
					conn = nil
				}
			}
		case "send":
			{
				if conn == nil {
					noConnMsg(tm)
					continue
				}

				if len(*fUser) > 0 {
					fmt.Fprintf(conn, "SEND %s: %s\n", *fUser, txt)
				} else {
					fmt.Fprintf(conn, "SEND %s\n", txt)
				}

				scn := bufio.NewScanner(conn)
				scn.Scan()
				tprint(scn.Text())
			}
		case "hist":
			{
				if conn == nil {
					noConnMsg(tm)
					continue
				}

				fmt.Fprintln(conn, "HIST")

				msgs, ind, err := getMsgList(conn)
				if err != nil {
					tprint(err.Error())
					continue
				}

				tprintf("%d, %d\n", len(msgs), ind)

				for _, msg := range msgs {
					tprint(msg)
				}
			}
		case "last":
			{
				if conn == nil {
					noConnMsg(tm)
					continue
				}

				if len(args) != 1 {
					tprint("'last' expects one argument")
					continue
				}

				n, err := strconv.Atoi(args[0])
				if err != nil {
					tprint(err.Error())
					continue
				}

				fmt.Fprintf(conn, "LAST %d\n", n)

				msgs, ind, err := getMsgList(conn)
				if err != nil {
					tprint(err.Error())
					continue
				}

				tprintf("%d, %d\n", len(msgs), ind)

				for _, msg := range msgs {
					tprint(msg)
				}
			}
		case "poll":
			{
				if conn == nil {
					noConnMsg(tm)
					continue
				}

				if len(args) != 1 {
					tprint("'poll' expects one argument")
					continue
				}

				n, err := strconv.Atoi(args[0])
				if err != nil {
					tprint(err.Error())
					continue
				}

				fmt.Fprintf(conn, "POLL %d", n)

				scn := bufio.NewScanner(conn)
				scn.Scan()
				tprint(scn.Text())
			}
		case "skip":
			{
				if conn == nil {
					noConnMsg(tm)
					continue
				}

				if len(args) != 1 {
					tprint("'skip' expects one argument")
					continue
				}

				n, err := strconv.Atoi(args[0])
				if err != nil {
					tprint(err.Error())
					continue
				}

				fmt.Fprintf(conn, "SKIP %d", n)

				scn := bufio.NewScanner(conn)

				scn.Scan()
				scn.Scan()
				tprint(scn.Text())

				scn.Scan()
				tprint(scn.Text())
			}
		case "stat":
			{
				fmt.Fprintln(conn, "STAT")

				scn := bufio.NewScanner(conn)

				scn.Scan()
				a := scn.Text()
				scn.Scan()
				b := scn.Text()
				scn.Scan()
				c := scn.Text()

				tprintf("%s, %s, %s\n", a, b, c)
			}
		default:
			tprintf("Unknown command '%s'\n", cmd)
		}
	}

	return nil
}

func main() {
	if err := _main(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
