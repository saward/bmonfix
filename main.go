package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"gopkg.in/yaml.v2"
)

// configurations.  Different monitor layouts are provided dependent upon the
// monitors available.  If a match is found to the monitors, that configuration
// is applied

// Layout A particular monitor's layout
type Layout struct {
	Monitor  string   `yaml:"monitor"`
	Desktops []string `yaml:"desktops"`
}

// Configuration A configuration for a particular monitor setup
type Configuration struct {
	Name     string   `yaml:"name"`
	Monitors []string `yaml:"monitors"`
	Layouts  []Layout `yaml:"layouts"`
}

// Configurations Array of all available configurations
type Configurations struct {
	Configurations []Configuration `yaml:"configurations"`
}

func main() {
	// Load configuration from file:
	cb, err := ioutil.ReadFile("configuration.yaml")
	if err != nil {
		log.Fatal(err)
	}

	var t Configurations

	err = yaml.Unmarshal([]byte(cb), &t)

	if err != nil {
		log.Fatal(err)
	}

	// Fetch the config relevant to this setup, returning error if none found
	ac, err := getActiveConfig(t)
	if err != nil {
		log.Fatal(err)
	}

	spew.Dump(ac)

	// Found a matching config, so let's apply it.  Steps:
	// 1. Move all desktops to appropriate monitors
	// 2. (?) Order those desktops
	// 3. (?) Delete unused

	for _, l := range ac.Layouts {
		for _, d := range l.Desktops {
			// Move desktop to specified monitor
			cmd := exec.Command("bspc", "desktop", d, "-m", l.Monitor)

			err := cmd.Run()

			if err != nil {
				// We note but ignore errors, to get something good enough:
				log.Print(err)
			}
		}
	}
}

// Returns array of values as returned from specified command
func getQueryList(command string, args ...string) ([]string, error) {
	var list []string

	cmd := exec.Command(command, args...)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()

	if err != nil {
		return list, err
	}

	trimmed := strings.TrimSpace(out.String())

	list = strings.Split(trimmed, "\n")

	return list, nil
}

func getMonitorsList() ([]string, error) {
	var monitors []string
	queryList, err := getQueryList("bspc", "query", "-M", "--names")

	if err != nil {
		return monitors, err
	}

	return queryList, err
}

// getActiveConfig Checks whether any available configs match the current
// monitor layout, returning that configuration if there's a match, otherwise
// returning an error
func getActiveConfig(configs Configurations) (Configuration, error) {
	var c Configuration
	monitors, err := getMonitorsList()

	if err != nil {
		return c, err
	}

	// Loop over our configs, and check if any of them matches the monitor layout
	// (order of monitors does not matter)

	found := false
	for _, c = range configs.Configurations {
		if len(c.Monitors) == len(monitors) {
			unmatched := len(c.Monitors)

			for _, em := range c.Monitors {
				for _, fm := range monitors {
					if em == fm {
						unmatched--
					}
				}
			}

			if unmatched == 0 {
				found = true
			}

			if found {
				break
			}
		}
	}

	if !found {
		return c, fmt.Errorf("Could not find any config for available monitor configuration")
	}

	return c, nil
}
