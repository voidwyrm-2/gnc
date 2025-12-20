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
	conf, confPath, err := getConfig()
	if err != nil {
		return err
	}

	version = strings.TrimSpace(version)

	fVersion := flag.Bool("v", false, "Show the current GNC version.")

	flag.Parse()

	if *fVersion {
		fmt.Println("Golang Nanochat Client (GNC), version", version)
		return nil
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	defer term.Restore(int(os.Stdin.Fd()), oldState)

	tm := term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}, "> ")

	var (
		addr   string
		conn   net.Conn
		buffer strings.Builder
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

	if len(conf.Username) == 0 {
		tprint("Warning: username is empty")
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
 config - shows the config path and the descriptions of the config options.
 conn [host] [port] - connects to the specified host and port.
 send <msg> - sends a message.
 hist - lists the previously sent messages.
 last <n> - show the specified number of previously sent messages.
 poll <n> - show the amount of messages after the specified index.
 skip <n> - gets the earliest messages after the specified index.
 add <msg> - appends text to the message buffer.
 sendbuf - sends the message buffer.
 showbuf - shows the contents of the message buffer.
 clearbuf - clears the buffer contents.
`, version)
		case "config":
			tprintf(`Config Path:
 '%s'

Config Options:
 Username - The username to use when connected to a Nanochat server.
 BufferAddSep - The separator to add to the buffer before adding the text from runnning 'add'.
 EntryMsg - The message to send after joined a server; '%name' will get replaced with the value of Username.
 ClearBufferOnSend - Should the buffer be cleared when running 'sendbuf'.
 DefaultHost - The default host to use when 'conn' is run with zero arguments.
 DefaultPort - The default port to use when 'conn' is run with zero or one arguments
`, confPath)
		case "conn":
			{
				if len(args) > 2 {
					tprint("'conn' expects zero, one, or two arguments")
					continue
				}

				var host, port string

				switch len(args) {
				case 0:
					host, port = conf.DefaultHost, conf.DefaultPort
				case 1:
					host, port = args[0], conf.DefaultPort
				case 2:
					host, port = args[0], args[1]
				}

				newAddr := fmt.Sprintf("%s:%s", host, port)

				if newAddr == addr {
					tprintf("Already connected to '%s'\n", newAddr)
				} else {
					if conn != nil {
						tprintf("Disconnecting from '%s'\n", addr)
						fmt.Fprintln(conn, "QUIT")
						conn.Close()
					}

					addr = newAddr
					tprintf("Opening a connection to '%s'\n", addr)

					conn, err = net.Dial("tcp", addr)
					if err != nil {
						tprint(err.Error())
						conn = nil
					}

					if len(conf.EntryMsg) > 0 {
						fmt.Fprintf(conn, "SEND %s\n", strings.ReplaceAll(conf.EntryMsg, "%name", conf.Username))
					}
				}
			}
		case "send":
			{
				if conn == nil {
					noConnMsg(tm)
					continue
				}

				if len(conf.Username) > 0 {
					fmt.Fprintf(conn, "SEND %s: %s\n", conf.Username, txt)
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

				fmt.Fprintf(conn, "POLL %d\n", n)

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

				fmt.Fprintf(conn, "SKIP %d\n", n)

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
		case "add":
			{
				if buffer.Len() > 0 && len(txt) > 0 {
					buffer.WriteString(conf.BufferAddSep)
				}

				buffer.WriteString(txt)
			}
		case "showbuf":
			tprintf("Buffer: `%s`\n", buffer.String())
		case "clearbuf":
			buffer.Reset()
		case "sendbuf":
			{
				if conn == nil {
					noConnMsg(tm)
					continue
				}

				txt := buffer.String()

				if len(conf.Username) > 0 {
					fmt.Fprintf(conn, "SEND %s: %s\n", conf.Username, txt)
				} else {
					fmt.Fprintf(conn, "SEND %s\n", txt)
				}

				if conf.ClearBufferOnSend {
					buffer.Reset()
				}

				scn := bufio.NewScanner(conn)
				scn.Scan()
				tprint(scn.Text())
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
