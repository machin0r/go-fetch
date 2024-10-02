# go-fetch

go-fetch is a system information fetching tool implemented in Go. It provides detailed hardware and software information about the host system.

## Features

- Displays system information including:
  - Hostname
  - OS and kernel version
  - Uptime
  - Shell
  - CPU details
  - GPU information
  - Memory usage
  - Package count (for supported Linux distributions)

## Dependencies

go-fetch relies on the following external libraries:

- github.com/jaypipes/ghw
- github.com/muesli/termenv
- github.com/shirou/gopsutil/v3

Ensure you have these dependencies installed before building the project.

## Building

To build the project, navigate to the project directory and run:

```
go build
```

This will create an executable named `go-fetch` in the current directory.

## Usage

Simply run the executable:

```
./go-fetch
```

The program will display system information in the terminal.