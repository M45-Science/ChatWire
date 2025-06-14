# Example Scripts

This directory provides small helper scripts for managing multiple ChatWire instances. They expect directories named `cw-a` through `cw-r` in your home directory.

- **example-build.sh** – build ChatWire and the agent then deploy the binaries to each target directory.
- **example-startup-script.sh** – sample invocation showing how to start ChatWire, with optional proxy or command registration.
- **example-start-factorio.sh** – create a `.start` file in each server directory to start Factorio.
- **example-stop-factorio.sh** – create a `.stop` file in each server directory to stop Factorio.
- **example-queue-reboot-factorio.sh** – create `.queue` files to queue Factorio reboots.
- **example-reboot-chatwire.sh** – create `.rebootcw` files to restart ChatWire instances.
- **make-services.sh** – generate systemd service and timer files for ChatWire and the agent.

Adjust the scripts as needed for your environment.

