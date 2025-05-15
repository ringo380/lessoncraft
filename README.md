# LessonCraft

LessonCraft is a web-based e-learning environment built to provide a simple and straightforward way to build and experience e-learning lessons that utilize a customized markdown syntax alognside interactive, live environments dynamically created docker containers and clusters.

A live deployment of LessonCraft is coming soon. Stay tuned for updates!

### Requirements

* [Docker `18.06.0+`](https://docs.docker.com/install/)
* [Go](https://golang.org/dl/) (stable release)

### Development

```bash
# Clone this repo locally
git clone https://github.com/ringo380/lessoncraft
cd lessoncraft

# Verify the Docker daemon is running
docker run hello-world

# Load the IPVS kernel module. Because swarms are created in dind,
# the daemon won't load it automatically
sudo modprobe xt_ipvs

# Ensure the Docker daemon is running in swarm mode
docker swarm init

# Get the latest franela/dind image
docker pull franela/dind

# Optional (with go1.14): pre-fetch module requirements into vendor
# so that no network requests are required within the containers.
# The module cache is retained in the pwd and l2 containers so the
# download is a one-off if you omit this step.
go mod vendor

# Start LessonCraft as a container
docker-compose up
```

Navigate to [http://localhost](http://localhost) and click "Start" to begin a new LessonCraft session, followed by "ADD NEW INSTANCE" to launch a new terminal instance.

### Port forwarding

In order for port forwarding to work correctly in development you need to make `*.localhost` to resolve to `127.0.0.1`. That way when you try to access  `pwd10-0-0-1-8080.host1.localhost`, then you're forwarded correctly to your local LessonCraft server.

You can achieve this by setting up a `dnsmasq` server (you can run it in a docker container also) and adding the following configuration:

```
address=/localhost/127.0.0.1
```

Don't forget to change your computer's default DNS to use the dnsmasq server to resolve.

## FAQ

### Why is LessonCraft running in ports 80 and 443? Can I change that?

No, it needs to run on those ports for DNS resolve to work. Ideas or suggestions about how to improve this
are welcome

## Hints

### How can I use Copy / Paste shortcuts?

Ctrl  + insert  : Copy
Shift + insert  : Paste
