package main

import (
	"encoding/json"
	"flag"
	"fmt"
	//"math"
	"os"
	//"os/exec"
	"os/signal"
	//"sort"
	"bytes"
	"errors"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gosuri/uilive"
	//"github.com/paulbellamy/ratecounter"
	"gopkg.in/redis.v4"
)

// Config ...
type Config struct {
	RedisHosts map[string][]string `json:"redis_hosts"`
}

var (
	refreshRate       int
	valLastTotal      int64
	valCurrTotal      int64
	maxColWidth       int
	maxColWidthMedium int

	buildDate          string
	version            string
	commitHash         string
	lastRedisHostStats map[string]map[string]int64
	currRedisHostStats map[string]map[string]int64
)

func main() {

	v := flag.Bool("v", false, "prints current version and exits")
	refreshRate := flag.Int("r", 2, "Referesh rate (seconds)")
	conf := flag.String("c", "", "Config containing the redis hosts along with the lists to monitor")
	flag.Parse()

	if *v {
		fmt.Printf("Version %s (commit: %s, %s)\n", version, commitHash, buildDate)
		os.Exit(0)
	}

	valLastTotal = 0
	valCurrTotal = 0
	maxColWidth = 38
	maxColWidthMedium = 26

	// Record number of events added/removed per list per: 10s, 1m, 5m

	/*
		rateCounterLast10Seconds := ratecounter.NewRateCounter(10 * time.Second)
		rateCounterLastMinute := ratecounter.NewRateCounter(1 * time.Minute)
		rateCounterLast5Minutes := ratecounter.NewRateCounter(5 * time.Minute)
	*/

	//counter.Rate() / 60

	writer := uilive.New()

	//ticker := time.Tick(time.Second)

	redisConf, err := loadConfig(*conf)
	if err != nil {
		exitWithMessage("ERROR", err.Error(), false)
	}
	redisClientConns := getRedisClientConns(redisConf)

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		writer.Stop()
		os.Exit(0)
	}()

	lastRedisHostStats = make(map[string]map[string]int64, len(redisConf.RedisHosts))
	currRedisHostStats = make(map[string]map[string]int64, len(redisConf.RedisHosts))

	writer.Start()

	var consoleOutput bytes.Buffer
	var diff int64
	var diffSign string
	firstItteration := true

	// Initialize the slize of redis host connections

	// ------------- Create a slice containing the redis hosts sorted -------------
	sortedRedisHosts := make([]string, len(redisConf.RedisHosts))
	i := 0
	for k, _ := range redisConf.RedisHosts {
		sortedRedisHosts[i] = k
		i++
	}
	sort.Strings(sortedRedisHosts)

	// -------------- Create another slice with the list of keys in sorted order for each redis host --------------
	sortedRedisLists := map[string][]string{}
	for rh, lists := range redisConf.RedisHosts {
		sort.Strings(lists)
		sortedRedisLists[rh] = lists
	}

	for {

		for h, c := range redisClientConns {
			// If the index of the given redis host isn't set, then set it
			if _, ok := currRedisHostStats[h]; !ok {
				currRedisHostStats[h] = make(map[string]int64, len(redisConf.RedisHosts[h]))
			}
			for _, list := range redisConf.RedisHosts[h] {
				// If the index of the given redis list isn't set, then set it
				if _, ok := currRedisHostStats[h][list]; !ok {
					currRedisHostStats[h][list] = 0
				}
				llen := c.LLen(list).Val()
				currRedisHostStats[h][list] = llen
			} // End of innner for loop
		} // End of outter for loop

		consoleOutput.WriteString(fmt.Sprintf("|%s|\n", strings.Repeat("-", 118)))
		consoleOutput.WriteString(fmt.Sprintf("|%s|%s|%s|%s|\n", center("Redis Hosts", 45), center("List", 40), center("Length", 14), center("Diff", 14)))
		consoleOutput.WriteString(fmt.Sprintf("|%s|\n", strings.Repeat("-", 118)))

		if !firstItteration {

			for _, rh := range sortedRedisHosts {
				for _, list := range sortedRedisLists[rh] {
					//for rh, lists := range currRedisHostStats {
					//for list, _ := range lists {

					/*
						1. If last reported list length is greater than this one, then it's a -DIFF (more consumed than put in)
						2  If last reported list length is smaller than this one, then it's a +DIFF (more put in than consumed)
					*/
					if lastRedisHostStats[rh][list] > currRedisHostStats[rh][list] {
						diff = lastRedisHostStats[rh][list] - currRedisHostStats[rh][list]
						diffSign = "-"
					} else if lastRedisHostStats[rh][list] < currRedisHostStats[rh][list] {
						diff = currRedisHostStats[rh][list] - lastRedisHostStats[rh][list]
						diffSign = "+"
					} else if lastRedisHostStats[rh][list] == 0 && currRedisHostStats[rh][list] == 0 {
						diff = 0
						diffSign = ""
					}
					consoleOutput.WriteString(fmt.Sprintf("|%s|%s|%s|%s|\n", rightAlign(rh, 45), rightAlign(list, 40), center(strconv.FormatInt(currRedisHostStats[rh][list], 10), 14), center(fmt.Sprintf("%s%d", diffSign, diff), 14)))

				}
				consoleOutput.WriteString(fmt.Sprintf("| %s |\n", strings.Repeat("-", 116)))
			}

		} else {
			firstItteration = false
		}

		consoleOutput.WriteString(fmt.Sprintf("|%s|\n", strings.Repeat("-", 118)))

		fmt.Fprintf(writer, "%s", fmt.Sprint(consoleOutput.String()))

		time.Sleep(time.Second * time.Duration(*refreshRate))
		consoleOutput.Reset()
		//sortedRedisHosts = sortedRedisHosts[:0]
	}

}

// Parse is a function that unmarshals the specified yaml config file
func (c *Config) Parse(data []byte) error {
	if err := json.Unmarshal(data, c); err != nil {
		return err
	}

	if len(c.RedisHosts) == 0 {
		return errors.New("Must have at least one redis host to monitor")
	}

	return nil
}

func getRedisClientConns(conf *Config) map[string]*redis.Client {
	clientConns := map[string]*redis.Client{}
	for h := range conf.RedisHosts {
		conn := redis.NewClient(&redis.Options{
			Addr: h,
			DB:   0,
		})
		clientConns[h] = conn
	}
	return clientConns
}

func loadConfig(filname string) (*Config, error) {

	content, err := ioutil.ReadFile(filname)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := config.Parse(content); err != nil {
		return nil, err
	}

	return &config, nil
}

func getRedisInfo(c *redis.Client) string {
	return c.Info().Val()
}

func exitWithMessage(level string, msg string, showUsage bool) {
	fmt.Printf("[%s] %s\n", level, msg)
	if showUsage {
		flag.PrintDefaults()
	}
	os.Exit(1)
}

func centerFill(s string, colWidth int, fill string) string {
	div := (colWidth - len(s)) / 2
	return strings.Repeat(fill, div) + s + strings.Repeat(fill, div)
}

func center(s string, colWidth int) string {
	div := (colWidth - len(s)) / 2
	return strings.Repeat(" ", div) + s + strings.Repeat(" ", div)
}

func leftAlign(s string, maxColWidth int) string {

	if len(s) >= maxColWidth-3 {
		return s[0:(maxColWidth-3)] + "..."
	}

	return " " + s + strings.Repeat(" ", (maxColWidth-len(s)))
}

func rightAlign(s string, maxColWidth int) string {
	/*
		maxPadding := maxColWidth - len(s)
		if len(s) > maxColWidth-3 {
			return s[0:(maxColWidth-1)] + "..." + strings.Repeat(" ", maxPadding)
		}
		return strings.Repeat(" ", (maxColWidth-len(s))) + s
	*/
	if len(s) >= maxColWidth-4 {
		return s[0:(maxColWidth-4)] + "..."
	}

	return strings.Repeat(" ", (maxColWidth-len(s))) + s + " "
}
