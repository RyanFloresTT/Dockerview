# Dockerview

A terminal user interface (TUI) for viewing and managing Docker containers, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

![Dockerview In Action](/Assets/dockerview.gif)

## Features

- List all containers - View both running and stopped containers
- Beautiful TUI - Clean, colorful interface with Lip Gloss styling
- Keyboard navigation - Vi-style keybindings for easy navigation
- Real-time refresh - Update container list on demand
- Full-screen mode - Uses alternate screen buffer like vim/htop
- Status indicators - Clear visual distinction between running and stopped containers

## Installation

### Prerequisites

- Go 1.24 or higher
- Docker installed and running
- Docker daemon accessible (Docker Desktop or Docker Engine)

### Install from source

```bash
bash git clone https://github.com/RyanFloresTT/dockerview.git 
cd dockerview 
go build -o dockerview
./dockerview
```


## Built With

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Style definitions for terminal
- [Docker Go SDK](https://github.com/docker/docker) - Docker API client

## Roadmap

- [x] Container start/stop/restart functionality
- [ ] View container logs
- [x] Container stats (CPU, memory usage)
- [ ] Filter containers by status
- [ ] Search functionality
- [ ] Container inspection details
- [ ] Multi-container operations

## Contributing

Contributions are welcome! Feel free to open issues or submit pull requests.

## License

MIT License - feel free to use this project however you'd like.

## Acknowledgments

This project was created to reduce manual tasks in managing Docker containers at work. It's a rewrite of an earlier version, now with better styling using Lip Gloss.