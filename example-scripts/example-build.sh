#!/bin/bash

if [ ! "$BASH_VERSION" ] ; then
        echo "Not launched with bash, re-running with bash."
        bash build.sh
    exit 1
fi

USER_NAME=$(whoami)

set -e  # Exit immediately on error
set -u  # Treat unset variables as errors

echo "Cleaning and building ChatWire..."
go clean
go build

echo "Deploying to targets..."
for dir in ~/cw-{a..r}; do
    # Create main directory if missing
    mkdir -p "$dir"

    # Create panel subdirectory if missing
    mkdir -p "$dir/panel"

    # Copy ChatWire binary
    cp -f ChatWire "$dir/ChatWire"

    # Copy template.html into panel subdirectory
    cp -f panel/template.html "$dir/panel/template.html"

    echo "Deployed to $dir"
done

# Set permissions on all ChatWire binaries
chmod 775 /home/$USER_NAME/cw-*/ChatWire

echo "Deployment complete."

echo "Cleaning and building agent..."
cd agent
go clean
go build
echo "Deploying to targets..."
for dir in ~/cw-{a..r}; do
    # Create dir
   mkdir -p "$dir/agent"

    # Copy agent binary
    cp -f agent "$dir/agent/agent"

    echo "Deployed to $dir"
done

# Set permissions on all ChatWire binaries
chmod 775 /home/$USER_NAME/cw-*/agent/agent
