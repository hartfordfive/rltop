package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gosuri/uilive"
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
	debug             bool

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
	debug := flag.Bool("d", false, "Enable debug mode (write cpu profile to file)")
	flag.Parse()

	if *v {
		fmt.Printf("Version %s (commit: %s, %s)\n", version, commitHash, buildDate)
		os.Exit(0)
	}

	if *debug {
		f, err := os.Create("rltop.cpuprofile")
		if err != nil {
			log.Fatal("Could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("Could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	valLastTotal = 0
	valCurrTotal = 0
	maxColWidth = 38
	maxColWidthMedium = 26

	writer := uilive.New()

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
		if *debug {
			pprof.StopCPUProfile()
		}
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

		// Get the list lengths for the current itteration
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

		consoleOutput.WriteString(fmt.Sprintf("|%124s|\n", strings.Repeat("-", 123)))
		consoleOutput.WriteString(fmt.Sprintf("|%-50s|%31s|%22s|%18s|\n", " Redis Host ", " List ", " Length ", " Diff "))
		consoleOutput.WriteString(fmt.Sprintf("|%124s|\n", strings.Repeat("-", 123)))

		if !firstItteration {

			for _, rh := range sortedRedisHosts {
				for _, list := range sortedRedisLists[rh] {
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

					consoleOutput.WriteString(fmt.Sprintf("|%-50s|%31s|%22s|%18s|\n", " "+rh, list, strconv.FormatInt(currRedisHostStats[rh][list], 10), fmt.Sprintf("%s%d", diffSign, diff)))

				}
				consoleOutput.WriteString(fmt.Sprintf("|%124s|\n", strings.Repeat("-", 123)))
			}

		} else {
			firstItteration = false
		}

		fmt.Fprintf(writer, "%s", fmt.Sprint(consoleOutput.String()))

		/* Now set the last redis list length values */
		for h, lists := range currRedisHostStats {
			if _, ok := lastRedisHostStats[h]; !ok {
				lastRedisHostStats[h] = make(map[string]int64, len(redisConf.RedisHosts[h]))
			}
			for list := range lists {
				if _, ok := lastRedisHostStats[h][list]; !ok {
					lastRedisHostStats[h][list] = 0
				}
				lastRedisHostStats[h][list] = currRedisHostStats[h][list]
			}
		}

		time.Sleep(time.Second * time.Duration(*refreshRate))
		consoleOutput.Reset()
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
