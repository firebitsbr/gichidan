// Copyright 2017 hIMEI

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/hIMEI29A/gotorsocks"
	"golang.org/x/net/html"
)

// Spider is an async urls handler
type Spider struct {
	// Urls already being handled
	HandledUrls map[string]bool
}

// NewSpider is a constructor for Spider
func NewSpider() *Spider {
	handled := make(map[string]bool)
	spider := &Spider{}
	spider.HandledUrls = handled

	return spider
}

// RequestProvider creates GET request with given string
func requestProvider(request string) string {
	var fullRequest string

	switch {
	case string(request[0]) == "/":
		fullRequest = "GET " + request + "\n"

	case string(request[0]) != "/" &&
		string(request[0]) != NONE &&
		strings.Contains(request, NONE) == true:

		splitted := strings.Split(request, NONE)
		fullRequest = "GET " + SEARCH + splitted[0] + "\n"

	default:
		fullRequest = "GET " + SEARCH + request + "\n"
	}

	return fullRequest
}

// ConnectProvider provides connect to Ichidan with gotorsocks package
func connectProvider() net.Conn {
	tor, err := gotorsocks.NewTorGate()
	ErrFatal(err)

	connect, err := tor.DialTor(ICHIDAN)
	ErrFatal(err)

	return connect
}

// GetContents makes request to Ichidan search engine and gets response body
func getContents(request string) chan *html.Node {
	chanNode := make(chan *html.Node)
	go func() {
		connect := connectProvider()
		defer connect.Close()

		fmt.Fprintf(connect, request)
		resp := bufio.NewReader(connect)

		node, err := htmlquery.Parse(resp)
		ErrFatal(err)
		chanNode <- node
	}()

	return chanNode
}

// GetTotal gets results total number
func (s *Spider) getTotal(root *html.Node) string {
	total := trimString(getTag(root, TOTAL))

	return total
}

// CheckResult controls empty search results
func (s *Spider) checkResult(node *html.Node) bool {
	ch := true

	result := findEntry(node, NORESULT)
	if result != nil {
		ch = false
	}

	return ch
}

// CheckRoot checks if given page is first or single page
func (s *Spider) checkRoot(node *html.Node) bool {
	check := false

	if s.checkSingle(node) == false || getTag(findEntry(node, PAGINATION), CURRENT) == "1" {
		check = true
	}

	return check
}

// CheckDone checks last pagination's page
func (s *Spider) checkDone(node *html.Node) bool {
	check := false

	pagination := findEntry(node, PAGINATION)

	if findEntry(pagination, DISABLED) != nil {
		check = true
	}

	return check
}

// CheckSingle checks if given page is single (have not pagination)
func (s *Spider) checkSingle(node *html.Node) bool {
	check := true

	if findEntry(node, PAGINATION) == nil {
		check = false
	}

	return check
}

// Crawl is a async crawler that takes request as first argument, gets it content
// and sends it to channel given as second argument
func (s *Spider) Crawl(url string, channelBody chan *html.Node /*, wg *sync.WaitGroup*/) {
	chanNode := getContents(url)
	body := <-chanNode

	if s.checkResult(body) == false {
		errString := BOLD + RED + "Nothing found there, Neo!" + RESET
		err := errors.New(errString)
		ErrFatal(err)
	}

	channelBody <- body

	return
}

// GetPagination finds pagination <div> and gets all links from it.
// Also it checks for single-paged result
func (s *Spider) getPagination(node *html.Node, chanUrls chan string) {
	pagination := findEntry(node, PAGINATION)

	if pagination != nil {
		current := toInt(getTag(pagination, CURRENT))
		hrefs := findEntrys(pagination, LINK)

		for _, newtag := range hrefs {
			if htmlquery.InnerText(newtag) != PREVIOUS &&
				htmlquery.InnerText(newtag) != NEXT &&
				toInt(htmlquery.InnerText(newtag)) > current {
				link := requestProvider(getHref(newtag))

				chanUrls <- link
			}
		}
	} else {
		fmt.Println(BOLD, GRN, "Only one page", RESET)
	}

	return
}
