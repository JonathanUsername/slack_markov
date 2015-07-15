// The Markov chain section of this code is taken from the "Generating arbitrary text"
// codewalk that can be found here: http://golang.org/doc/codewalk/markov/

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jmoiron/jsonq"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"strings"
)

var DEFAULT_WORD_LENGTH int = 2000
var DEFAULT_PREFIX_LENGTH int = 2
var STRIP_TAGS bool = false

// Prefix is a Markov chain prefix of one or more words.
type Prefix []string

// String returns the Prefix as a string (for use as a map key).
func (p Prefix) String() string {
	return strings.Join(p, " ")
}

// Shift removes the first word from the Prefix and appends the given word.
func (p Prefix) Shift(word string) {
	copy(p, p[1:])
	p[len(p)-1] = word
}

// Chain contains a map ("chain") of prefixes to a list of suffixes.
// A prefix is a string of prefixLen words joined with spaces.
// A suffix is a single word. A prefix can have multiple suffixes.
type Chain struct {
	chain     map[string][]string
	prefixLen int
}

// NewChain returns a new Chain with prefixes of prefixLen words.
func NewChain(prefixLen int) *Chain {
	return &Chain{make(map[string][]string), prefixLen}
}

// Build reads text from the provided Reader and
// parses it into prefixes and suffixes that are stored in Chain.
func (c *Chain) Build(r io.Reader) {
	br := bufio.NewReader(r)
	p := make(Prefix, c.prefixLen)
	for {
		var s string
		if _, err := fmt.Fscan(br, &s); err != nil {
			break
		}
		key := p.String()
		// fmt.Println(key)
		c.chain[key] = append(c.chain[key], s)
		p.Shift(s)
	}
}

// Generate returns a string of at most n words generated from Chain.
func (c *Chain) Generate(n int, single bool) string {
	p := make(Prefix, c.prefixLen)
	var words []string
	var possible_prefixes []string
	// select first word randomly
	for each, _ := range c.chain {
		if len(each) > 6 && each[:6] == "<end/>" && !strings.Contains(each[6:], "<end/>") {
			possible_prefixes = append(possible_prefixes, each)
		}
	}
	var first_prefix string
	first_prefix = possible_prefixes[rand.Intn(len(possible_prefixes))]
	arr := strings.Split(first_prefix, " ")
	for i := 0; i < c.prefixLen; i++ {
		p[i] = arr[i]
	}
	words = append(words, first_prefix)
	// generate it
	for i := 0; i < n; i++ {
		var word string = p.String()
		choices := c.chain[word]
		if len(choices) == 0 {
			// fmt.Println("No choices. Breaking.")
			break
		}
		next := choices[rand.Intn(len(choices))]
		if single && strings.Contains(next, "<end/>") {
			// fmt.Println("Found end tag. Breaking.")
			break
		}
		words = append(words, next)
		if i == n-1 {
			// fmt.Println("Reached word limit. Breaking.")
			words = append(words, "[...]")
		}
		p.Shift(next)
	}
	return strings.Join(words, " ")[6:]
}

func parseScrape(stringbody string, user string) string {
	results, err := fromJson(stringbody).Array("messages")
	check(err)
	var wholebody string
	for _, result := range results {
		if checkUser(result, user) {
			if body, ok := result.(map[string]interface{})["text"]; ok {
				var chunk string = body.(string)
				chunk += " <end/>"
				wholebody += chunk
			}
		}
	}
	return wholebody
}

func checkUser(result interface{}, user string) bool {
	if result.(map[string]interface{})["user"] == user || user == "" {
		return true
	}
	return false
}

func buildPost(user string, scrape string, prefixL int, wordL int, single_sentence bool) string {
	// fmt.Println("Building " + user)
	c := NewChain(prefixL)
	cleanbody := parseScrape(scrape, user)
	b := io.Reader(strings.NewReader(cleanbody))
	c.Build(b)
	text := c.Generate(wordL, single_sentence)
	var rx string
	if STRIP_TAGS {
		rx = "<[^>]*>"
	} else {
		rx = "<end/>"
	}
	strip_tags_regex, _ := regexp.Compile(rx)
	cleantext := strip_tags_regex.ReplaceAllString(text, "")
	return cleantext
}

func main() {
	var user string
	var userid string
	flag.IntVar(&DEFAULT_PREFIX_LENGTH, "prefix", 2, "Prefix length for chain creation.")
	flag.IntVar(&DEFAULT_WORD_LENGTH, "max-words", 100, "Maximum word length for output.")
	flag.BoolVar(&STRIP_TAGS, "no-tags", false, "Whether to strip html tags from input.")
	flag.StringVar(&user, "user", "", "User name.")
	flag.Parse()
	if user == "" {
		fmt.Println("Missing --user flag for user name.")
		os.Exit(1)
	}
	bytes, err := ioutil.ReadAll(os.Stdin)
	check(err)
	scrape := string(bytes)
	userid = getId(user)
	resp := buildPost(userid, scrape, DEFAULT_PREFIX_LENGTH, DEFAULT_WORD_LENGTH, false)
	fmt.Println(resp)
}

func getId(name string) string {
	var id string
	// A file generated with "curl https://slack.com/api/users.list?token=$TOKEN > userslist.json"
	file, err := ioutil.ReadFile("userslist.json")
	check(err)
	categories, err := fromJson(string(file)).Array("members")
	check(err)
	for _, entry := range categories {
		if entry.(map[string]interface{})["name"] == name {
			id = entry.(map[string]interface{})["id"].(string)
			return id
		}
	}
	panic("Cannot find user id!")
}

func fromJson(js string) *jsonq.JsonQuery {
	// usage: var, err := fromJson(json).String("value", "nestedvalue", "somearray, "0")
	data := map[string]interface{}{}
	dec := json.NewDecoder(strings.NewReader(js))
	dec.Decode(&data)
	jq := jsonq.NewQuery(data)
	return jq
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
