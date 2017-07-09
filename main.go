package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func main() {
	monitors, err := GetQueryList("bspc", "query", "-M")

	if err != nil {
		panic(err)
	}

	desktops, err := GetQueryList("bspc", "query", "-D")

	if err != nil {
		panic(err)
	}

	monitorCount := 0
	desktopCount := 0

	for desktopCount < (len(desktops) - 1) {
		fmt.Printf("%d (%d): bspc desktop %s -m %s\n", desktopCount, monitorCount, desktops[desktopCount], monitors[monitorCount])

		cmd := exec.Command("bspc", "desktop", desktops[desktopCount], "-m", monitors[monitorCount])

		err := cmd.Run()

		if err != nil {
			log.Print(err)
		}

		desktopCount++
		if desktopCount >= (len(desktops)/len(monitors))*(monitorCount+1) {
			monitorCount++
		}
	}

	// Remove the first desktop on monitors past the first:

	if len(monitors) > 1 {
		count := 1

		for count < len(monitors) {
			desktops, err := GetQueryList("bspc", "query", "-m", monitors[count], "-D")

			if err != nil {
				log.Print("Error getting list of desktops for monitor %d (%s): %s", count, monitors[count], err)
				continue
			}

			if len(desktops) > 10 {
				first, err := GetMonitorFirstDesktop(monitors[count])
				if err == nil {
					cmd := exec.Command("bspc", "desktop", first, "-r")

					err := cmd.Run()

					if err != nil {
						log.Print("Error deleting first desktop on monitor %d (%s): %s", count, monitors[count], err)
					}
				} else {
					log.Print("Error getting first desktop for monitor %d (%s): %s", count, monitors[count], err)
				}
			}

			count++
		}
	}
}

func GetQueryList(command string, args ...string) ([]string, error) {
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

func GetMonitorFirstDesktop(monitor string) (string, error) {
	queryList, err := GetQueryList("bspc", "query", "-m", monitor, "-D")

	if err != nil {
		return "", err
	}

	return queryList[0], nil
}
