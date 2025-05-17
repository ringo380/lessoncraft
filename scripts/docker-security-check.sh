#!/bin/bash
set -e

# Docker Security Check Script
# This script performs security checks on Docker images and configurations

echo "=== LessonCraft Docker Security Check ==="
echo

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed or not in PATH"
    exit 1
fi

# Check if Docker Scout is installed
if ! command -v docker-scout &> /dev/null; then
    echo "Warning: Docker Scout is not installed. Vulnerability scanning will be skipped."
    echo "Install Docker Scout: https://docs.docker.com/scout/install/"
    SKIP_SCOUT=true
else
    SKIP_SCOUT=false
fi

echo "Building Docker images..."
echo

# Build the main LessonCraft image
echo "Building main LessonCraft image..."
docker build -t lessoncraft:local -f Dockerfile .

# Build the L2 image
echo "Building L2 image..."
docker build -t lessoncraft-l2:local -f Dockerfile.l2 .

# Build the DinD image
echo "Building DinD image..."
docker build -t lessoncraft-dind:local -f dockerfiles/dind/Dockerfile dockerfiles/dind

echo
echo "Running security checks..."
echo

# Check for root user in images
echo "Checking for non-root users in images..."
LESSONCRAFT_USER=$(docker run --rm lessoncraft:local id -u)
if [ "$LESSONCRAFT_USER" = "0" ]; then
    echo "Warning: LessonCraft image is running as root"
else
    echo "OK: LessonCraft image is running as non-root user (UID: $LESSONCRAFT_USER)"
fi

# L2 image might need to run as root for port binding
echo "Note: L2 image may need to run as root for port binding (22, 53)"

# Check for Docker Content Trust
echo "Checking Docker Content Trust setting..."
if [ -z "$DOCKER_CONTENT_TRUST" ]; then
    echo "Warning: DOCKER_CONTENT_TRUST is not set. Consider enabling it for production."
else
    echo "OK: DOCKER_CONTENT_TRUST is set to $DOCKER_CONTENT_TRUST"
fi

# Check Docker daemon configuration
echo "Checking Docker daemon configuration..."
if [ -f /etc/docker/daemon.json ]; then
    if grep -q "\"experimental\": true" /etc/docker/daemon.json; then
        echo "Warning: Docker daemon has experimental features enabled"
    fi
    
    if grep -q "\"debug\": true" /etc/docker/daemon.json; then
        echo "Warning: Docker daemon has debug mode enabled"
    fi
    
    if grep -q "\"insecure-registries\"" /etc/docker/daemon.json; then
        echo "Warning: Docker daemon has insecure registries configured"
    fi
    
    if grep -q "\"tcp://0.0.0.0" /etc/docker/daemon.json; then
        echo "Warning: Docker daemon is exposed on all interfaces"
    fi
else
    echo "Note: No custom Docker daemon configuration found"
fi

# Run Docker Scout vulnerability scanning if available
if [ "$SKIP_SCOUT" = "false" ]; then
    echo
    echo "Running vulnerability scans with Docker Scout..."
    
    echo "Scanning LessonCraft image..."
    docker scout cves lessoncraft:local
    
    echo "Scanning L2 image..."
    docker scout cves lessoncraft-l2:local
    
    echo "Scanning DinD image..."
    docker scout cves lessoncraft-dind:local
fi

echo
echo "=== Security Check Complete ==="
echo "Review any warnings or vulnerabilities and address them before deployment."