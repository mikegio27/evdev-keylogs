package main

import (
	"log"

	"github.com/mikegio27/go-evdev"
)

func main() {
	dataChanMap, cancel, err := evdev.MonitorDevices(nil)
	keyMap := evdev.GenerateKeyMap()
	if err != nil {
		log.Fatalf("Failed to monitor devices: %v", err)
	}
	defer cancel()

	// Handle the data from the channels
	for devicePath, dataChan := range dataChanMap {
		go func(devicePath string, dataChan chan evdev.InputEvent) {
			for event := range dataChan {
				log.Printf("Device: %s, Event: %+v\n", devicePath, keyMap[event.Code])
			}
		}(devicePath, dataChan)
	}

	// Keep the main function running
	select {}
}
