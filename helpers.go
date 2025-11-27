package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func getMsgList(conn io.ReadWriter) (msgs []string, ind int, err error) {
	scn := bufio.NewScanner(conn)
	scn.Scan()

	n, err := strconv.Atoi(scn.Text())
	if err != nil {
		return nil, -1, err
	}

	msgs = make([]string, 0, n)

	for range n {
		scn.Scan()
		msgs = append(msgs, strings.TrimSpace(scn.Text()))
	}

	scn.Scan()

	ind, err = strconv.Atoi(scn.Text())
	if err != nil {
		return nil, -1, err
	}

	return
}

func noConnMsg(conn io.Writer) {
	fmt.Fprintln(conn, "No open connection, please use 'conn' to create one")
}
