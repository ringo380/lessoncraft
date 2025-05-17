# LessonCraft

[![Go Tests](https://github.com/ringo380/lessoncraft/actions/workflows/test.yml/badge.svg)](https://github.com/ringo380/lessoncraft/actions/workflows/test.yml)
[![Docker Build and Publish](https://github.com/ringo380/lessoncraft/actions/workflows/docker-build.yml/badge.svg)](https://github.com/ringo380/lessoncraft/actions/workflows/docker-build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ringo380/lessoncraft)](https://goreportcard.com/report/github.com/ringo380/lessoncraft)
[![codecov](https://codecov.io/gh/ringo380/lessoncraft/branch/master/graph/badge.svg)](https://codecov.io/gh/ringo380/lessoncraft)

LessonCraft is a web-based e-learning environment built to provide a simple and straightforward way to build and experience e-learning lessons that utilize a customized markdown syntax alongside interactive, live environments dynamically created using Docker containers and clusters.

## Overview

LessonCraft enables educators to:

1. **Create Interactive Lessons**: Build lessons using a specialized Markdown format with code blocks for commands, expected outputs, and questions.
2. **Provide Live Environments**: Give learners access to real Docker environments where they can execute commands and see results in real-time.
3. **Validate Learning**: Automatically validate learner progress by comparing command outputs with expected results.

A live deployment of LessonCraft is coming soon. Stay tuned for updates!

## Getting Started

### Requirements

* [Docker `18.06.0+`](https://docs.docker.com/install/)
* [Go `1.24.0+`](https://golang.org/dl/) (stable release)
* MongoDB (for lesson storage)

### Development Setup

1. **Clone the repository**:
   ```bash
   git clone https://github.com/ringo380/lessoncraft
   cd lessoncraft
   ```

2. **Verify Docker is installed and running**:
   ```bash
   docker run hello-world
   ```

3. **Load the IPVS kernel module** (required for Docker swarm functionality):
   ```bash
   sudo modprobe xt_ipvs
   ```

4. **Initialize Docker swarm mode**:
   ```bash
   docker swarm init
   ```

5. **Pull the required Docker image**:
   ```bash
   docker pull franela/dind
   ```

6. **Configure environment variables** (optional):
   ```bash
   # Copy the sample environment file
   cp .env.sample .env

   # Edit the .env file to customize your configuration
   nano .env
   ```

7. **Prepare Go modules**:
   ```bash
   go mod tidy
   go mod vendor  # Optional: pre-fetch dependencies
   ```

8. **Start LessonCraft**:
   ```bash
   docker-compose up
   ```

9. **Access the application**:
   Navigate to [http://localhost](http://localhost) and click "Start" to begin a new LessonCraft session, followed by "ADD NEW INSTANCE" to launch a new terminal instance.

## Configuration

LessonCraft can be configured using environment variables. A sample configuration file (`.env.sample`) is provided as a starting point.

### Environment Variables

#### Host Configuration
- `HOST_PORT`: The port to expose on the host (default: 80)

#### Container Names
- `HAPROXY_CONTAINER_NAME`: Name for the HAProxy container (default: haproxy)
- `LESSONCRAFT_CONTAINER_NAME`: Name for the main application container (default: lessoncraft)
- `L2_CONTAINER_NAME`: Name for the L2 networking container (default: l2)
- `MONGODB_CONTAINER_NAME`: Name for the MongoDB container (default: mongodb)

#### Service Configuration
- `MONGODB_URI`: MongoDB connection string (default: mongodb://mongodb:27017)
- `MONGODB_DATABASE`: MongoDB database name (default: lessoncraft)

#### Application Configuration
- `LESSONCRAFT_UNSAFE`: Enable unsafe mode (default: false)
- `PLAYGROUND_DOMAIN`: Domain for the playground (default: localhost)
- `DEFAULT_DIND_IMAGE`: Default Docker-in-Docker image (default: franela/dind)
- `AVAILABLE_DIND_IMAGES`: Available Docker-in-Docker images (default: franela/dind)
- `ALLOW_WINDOWS_INSTANCES`: Allow Windows instances (default: false)
- `DEFAULT_SESSION_DURATION`: Default session duration (default: 4h)
- `MAX_LOAD_AVG`: Maximum allowed load average (default: 100)

#### Security Configuration
- `APPARMOR_PROFILE`: AppArmor profile for containers (default: docker-dind)
- `DOCKER_CONTENT_TRUST`: Enable Docker content trust (default: 1)
- `COOKIE_HASH_KEY`: Hash key for secure cookies
- `COOKIE_BLOCK_KEY`: Block key for secure cookies
- `ADMIN_TOKEN`: Token for admin endpoints

#### Ports
- `SSH_PORT`: Port for SSH access (default: 8022)
- `DNS_PORT`: Port for DNS service (default: 8053)
- `TLS_PORT`: Port for TLS connections (default: 443)

### Persistent Volumes

LessonCraft uses several Docker volumes for persistent data:

- `sessions`: Stores session data for the main application
- `networks`: Stores network configuration for the L2 container
- `data`: Stores application data for the main application
- `mongodb_data`: Stores MongoDB database files

You can customize the volume names using environment variables:
- `SESSIONS_VOLUME`: Name for the sessions volume (default: sessions)
- `NETWORKS_VOLUME`: Name for the networks volume (default: networks)
- `DATA_VOLUME`: Name for the data volume (default: data)
- `MONGODB_DATA_VOLUME`: Name for the MongoDB data volume (default: mongodb_data)

### Resource Limits

Resource limits are configured directly in the `docker-compose.yml` file:

- HAProxy: 256MB memory, 0.5 CPU cores
- LessonCraft: 512MB memory, 1.0 CPU cores
- L2: 512MB memory, 1.0 CPU cores
- MongoDB: 512MB memory, 0.5 CPU cores

### Running Tests

Run all tests:
```bash
go test ./...
```

Run tests with verbose output:
```bash
go test -v ./...
```

Run specific tests:
```bash
go test -v ./lesson
go test -v ./api
go test -v ./api/store
```

### Port Forwarding

In order for port forwarding to work correctly in development you need to make `*.localhost` resolve to `127.0.0.1`. That way when you try to access `pwd10-0-0-1-8080.host1.localhost`, you're forwarded correctly to your local LessonCraft server.

You can achieve this by setting up a `dnsmasq` server (you can run it in a docker container also) and adding the following configuration:

```
address=/localhost/127.0.0.1
```

Don't forget to change your computer's default DNS to use the dnsmasq server to resolve.

## API Documentation

LessonCraft provides a RESTful API for managing lessons:

### Lesson Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/lessons` | GET | List all lessons |
| `/api/lessons/{id}` | GET | Get a specific lesson by ID |
| `/api/lessons` | POST | Create a new lesson |
| `/api/lessons/{id}` | PUT | Update an existing lesson |
| `/api/lessons/{id}` | DELETE | Delete a lesson |

### Lesson Interaction

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/lessons/{id}/start` | POST | Start a lesson |
| `/api/lessons/{id}/steps/{step}/complete` | POST | Complete a step in a lesson |
| `/api/lessons/{id}/validate` | POST | Validate a step in a lesson |

For detailed API documentation, including request and response formats, see [docs/lesson_format.md](docs/lesson_format.md#api-usage).

## Lesson Format

LessonCraft uses a specialized Markdown format for creating interactive lessons. Lessons consist of:

1. **Title**: A level 1 heading (`# Title`)
2. **Description**: Text following the title
3. **Steps**: A series of steps, each containing content, commands, expected outputs, and/or questions

### Code Blocks

LessonCraft uses specialized code blocks:

- **Docker Blocks** (````docker`): Define commands to execute
- **Expect Blocks** (````expect`): Define expected command output
- **Question Blocks** (````question`): Define questions for the user

For a complete guide to the lesson format, see [docs/lesson_format.md](docs/lesson_format.md).

## Troubleshooting

### Common Issues

1. **Port conflicts**: Ensure ports 80 and 443 are available
   ```bash
   # Check if ports are in use
   sudo lsof -i :80
   sudo lsof -i :443
   ```

2. **Docker permissions**: Make sure your user has permissions to access the Docker socket
   ```bash
   # Add your user to the docker group
   sudo usermod -aG docker $USER
   # Then log out and back in
   ```

3. **DNS resolution**: For port forwarding, ensure `*.localhost` resolves to `127.0.0.1`
   ```bash
   # Test DNS resolution
   ping test.localhost
   # Should resolve to 127.0.0.1
   ```

4. **MongoDB connection**: Ensure MongoDB is running and accessible
   ```bash
   # Test MongoDB connection
   mongo --eval "db.version()"
   ```

### FAQ

#### Why is LessonCraft running on ports 80 and 443? Can I change that?

No, it needs to run on those ports for DNS resolution to work. Ideas or suggestions about how to improve this are welcome.

#### How can I use Copy/Paste shortcuts in the terminal?

- **Copy**: Ctrl + Insert
- **Paste**: Shift + Insert

#### How do I create my own lessons?

See the [docs/lesson_format.md](docs/lesson_format.md) guide and check out the example lessons in the [examples/lessons](examples/lessons) directory.
