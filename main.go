package main

import (
	"fmt"
	"log"

	"github.com/mikegio27/go-evdev"
)

func main() {
	keyMap := evdev.GenerateKeyMap()
	devices, err := evdev.InputDevices()
	if err != nil {
		fmt.Println("Error getting input devices:", err)
		return
	}

	keyboardDevices := []evdev.InputDevice{}
	for _, device := range devices {
		if device.IsKeyboard() {
			keyboardDevices = append(keyboardDevices, device)
		}
	}

	if len(keyboardDevices) == 0 {
		fmt.Println("No keyboard devices found.")
		return
	}

	dataChanMap, cancel, err := evdev.MonitorDevices(keyboardDevices)
	if err != nil {
		fmt.Println("Error monitoring devices:", err)
		return
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
