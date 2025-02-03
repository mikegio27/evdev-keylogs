# evdev-keylogs

This project is for research on `evdev` with Golang. It is intended for studying how to log input events globally using an `evdev` library.

## Overview

The project monitors all input devices and logs their events. It uses the `go-evdev` library to interact with the input devices.

## Prerequisites

- Go 1.23.5 or later
- `github.com/mikegio27/go-evdev` library

## Installation

1. Clone the repository:
    ```sh
    git clone https://github.com/mikegio27/evdev-keylogs.git
    ```
2. Navigate to the project directory:
    ```sh
    cd evdev-keylogs
    ```
3. Install the dependencies:
    ```sh
    go mod tidy
    ```

## Usage

Run the project:
```sh
go run main.go
```

## Disclaimer
The program will start monitoring all input devices and log their events to the console.

This project is for educational purposes only. Use it responsibly and ensure you have permission to log input events on the devices you monitor.