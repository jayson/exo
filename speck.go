package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

func speckStream(r io.ReadCloser) {
	defer r.Close()
	keep := regexp.MustCompile(`^\s*[\d-]`)

	br := bufio.NewReader(r)
	for {
		line, err := br.ReadBytes('\n')
		switch err {
		case io.EOF:
			return
		case nil:
			break
		default:
			log_error("unable to stream speck file: %v", err)
			return
		}
		if !keep.Match(line) {
			continue
		}
		parseSpeckLine(line)
	}
}

type exoSystem struct {
	x, y, z float64
	planets int
	name    string
}

func (e exoSystem) String() string {
    return fmt.Sprintf("<name: %s x: %v y: %v z: %v planets: %v>", e.name, e.x, e.y, e.z, e.planets)
}

func parseSpeckLine(line []byte) {
	parts := strings.Split(string(line), " ")
	var err error
	var g errorGroup
	s := new(exoSystem)

	s.x, err = strconv.ParseFloat(parts[0], 64)
	g.AddError(err)
	s.y, err = strconv.ParseFloat(parts[1], 64)
	g.AddError(err)
	s.z, err = strconv.ParseFloat(parts[2], 64)
	g.AddError(err)
	s.planets, err = strconv.Atoi(parts[3])
	g.AddError(err)

    s.name = strings.TrimSpace(strings.Join(parts[7:], " "))

	if g != nil {
		log_error("%v", g)
	}
	log_info("%v", s)
}
