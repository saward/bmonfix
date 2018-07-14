package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"

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

	//spew.Dump(ac)

	// Found a matching config, so let's apply it.  Steps:
	// 1. Create a default desktop on each monitor so that we can move the ones we want
	// 2. Create any specified desktops that don't exist
	// 2. Move all desktops to appropriate monitors
	// 4. Delete unused/default
	// 5. (?) Order those desktops

	// Create default desktop on each monitor, if it doesn't already exist on that monitor:
	for _, l := range ac.Layouts {
		dName := fmt.Sprintf("default-%s", l.Monitor)

		_, err := createUncreatedDesktop(dName, l.Monitor, true)

		if err != nil {
			log.Fatal(err)
		}
	}

	// Create any listed desktops that don't exist on appropriate desktop, or
	// move if they exist to their correct desktop
	for _, l := range ac.Layouts {
		for _, d := range l.Desktops {

			created, err := createUncreatedDesktop(d, l.Monitor, false)

			if err != nil {
				log.Print(err)
				continue
			}

			if !created {
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

	// Delete any listed desktops that shouldn't exist
	for _, l := range ac.Layouts {

		// Get list of desktops that exist on specified monitor
		desktops, err := getQueryList("bspc", "query", "-D", "--names", "-m", l.Monitor)

		if err != nil {
			log.Print(err)
			continue
		}

		// Check whether or not each found desktop should be there, and delete
		// if not:
		for _, found := range desktops {
			var shouldExist bool

			for _, d := range l.Desktops {
				if found == d {
					shouldExist = true
					break
				}
			}

			// Shouldn't exist, so delete from monitor:
			if !shouldExist {
				id, err := getDesktopID(found, l.Monitor)

				if err != nil {
					log.Print(err)
					continue
				}

				cmd := exec.Command("bspc", "desktop", id, "-r")

				err = cmd.Run()

				if err != nil {
					log.Print(err)
					continue
				}
			}
		}
	}
}

// getDesktopID Returns the desktop ID for the named desktop on the specified
// monitor
func getDesktopID(desktop string, monitor string) (string, error) {
	var id string
	// Get desktop names, and then desktop id's, because we assume order is the
	// same:

	desktopNames, err := getQueryList("bspc", "query", "--names", "-D", "-m", monitor)

	if err != nil {
		return id, err
	}

	desktopIDs, err := getQueryList("bspc", "query", "-D", "-m", monitor)

	if err != nil {
		return id, err
	}

	for i, name := range desktopNames {
		if name == desktop {
			return desktopIDs[i], nil
		}
	}

	return id, fmt.Errorf("Desktop %s not found on monitor %s", desktop, monitor)
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

// createUncreatedDesktop Creates the listed desktop on the listed monitor if
// that desktop does not exist.  If monitor is not provided, then only create
// if doesn't exist anywhere
func createUncreatedDesktop(desktop string, monitor string, monitorMatch bool) (bool, error) {
	exists := false

	exists, err := checkDesktopExists(desktop, monitor, monitorMatch)

	if err != nil {
		return exists, err
	}

	if !exists {
		cmd := exec.Command("bspc", "monitor", monitor, "-a", desktop)

		err := cmd.Run()

		if err != nil {
			log.Fatal(err)
		}
	}

	return exists, nil
}

// checkDesktopExists Checks if desktop exists.  If monitor is provided, checks
// only that desktop
func checkDesktopExists(desktop string, monitor string, monitorMatch bool) (bool, error) {
	var exists bool
	var desktops []string
	var err error

	if monitorMatch {
		desktops, err = getQueryList("bspc", "query", "-D", "--names", "-m", monitor)
	} else {
		desktops, err = getQueryList("bspc", "query", "-D", "--names")
	}

	if err != nil {
		return exists, err
	}

	for _, d := range desktops {
		if d == desktop {
			return true, nil
		}
	}

	return false, nil
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
