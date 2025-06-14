#!/bin/bash

# Re-run this script with bash if not already using bash
if [ -z "${BASH_VERSION:-}" ]; then
    exec bash "$0" "$@"
fi

# === CONFIG ===

# Prompt user for letter range
read -p "Enter letters to generate (default: a to r): " input_letters
if [[ -z "$input_letters" ]]; then
    LETTERS=({a..r})
else
    IFS=',' read -ra LETTERS <<< "$input_letters"
fi

USER_NAME=$(whoami)
BASE_DIR="/home/$USER_NAME"
SERVICE_DIR="services"
mkdir -p "$SERVICE_DIR"

# === SERVICE GENERATORS ===

generate_agent_service() {
  local letter=$1
  cat <<EOF > "$SERVICE_DIR/agent-$letter.service"
[Unit]
Description=agent-$letter
Wants=network-online.target
After=network.target network-online.target

[Service]
User=$USER_NAME
WorkingDirectory=$BASE_DIR/cw-$letter/agent/
ExecStart=./agent
Restart=always
StartLimitBurst=3
StartLimitInterval=1
RestartSec=1

[Install]
WantedBy=multi-user.target
EOF
}

generate_chatwire_service() {
  local letter=$1
  cat <<EOF > "$SERVICE_DIR/chatwire-$letter.service"
[Unit]
Description=chatwire-$letter
Wants=network-online.target
After=network.target network-online.target

[Service]
User=$USER_NAME
WorkingDirectory=$BASE_DIR/cw-$letter/
ExecStart=/bin/bash start.sh
Restart=always
StartLimitBurst=3
StartLimitInterval=1
RestartSec=1

[Install]
WantedBy=multi-user.target
EOF
}

# === TIMER GENERATORS ===

generate_agent_timer() {
  local letter=$1
  cat <<EOF > "$SERVICE_DIR/agent-$letter.timer"
[Unit]
Description=Delayed Start for agent-$letter

[Timer]
OnBootSec=1s

[Install]
WantedBy=basic.target
EOF
}

generate_chatwire_timer() {
  local letter=$1
  local delay=$2
  cat <<EOF > "$SERVICE_DIR/chatwire-$letter.timer"
[Unit]
Description=Delayed Start for chatwire-$letter

[Timer]
OnBootSec=${delay}s

[Install]
WantedBy=basic.target
EOF
}

# === GENERATION PROCESS ===

echo "Generating agent services and timers..."
for letter in "${LETTERS[@]}"; do
  generate_agent_service "$letter"
  generate_agent_timer "$letter"
done

echo "Generating chatwire services and timers..."
delay=6
for letter in "${LETTERS[@]}"; do
  generate_chatwire_service "$letter"
  generate_chatwire_timer "$letter" "$delay"
  delay=$((delay + 1))
done

echo "✅ All files written to ./$SERVICE_DIR"

# === INSTALL PROMPT ===

read -p "Install all services and timers to /etc/systemd/system/? [y/N]: " install_choice
if [[ "$install_choice" =~ ^[Yy]$ ]]; then
  sudo cp "$SERVICE_DIR"/*.service "$SERVICE_DIR"/*.timer /etc/systemd/system/
  echo "✅ Files installed to /etc/systemd/system/"

  echo "Reloading systemd daemon..."
  sudo systemctl daemon-reexec
  sudo systemctl daemon-reload
  echo "✅ systemd daemon reloaded."

  read -p "Enable all timers so they start at boot? [y/N]: " enable_choice
  if [[ "$enable_choice" =~ ^[Yy]$ ]]; then
    for letter in "${LETTERS[@]}"; do
      sudo systemctl enable "agent-$letter.timer"
      sudo systemctl enable "chatwire-$letter.timer"
    done
    echo "✅ All timers enabled."
  else
    echo "⏸ Timers were not enabled to start at boot."
  fi

  read -p "Start all timers now? [y/N]: " start_choice
  if [[ "$start_choice" =~ ^[Yy]$ ]]; then
    for letter in "${LETTERS[@]}"; do
      sudo systemctl start "agent-$letter.timer"
      sudo systemctl start "chatwire-$letter.timer"
    done
    echo "▶️ All timers started."
  else
    echo "⏸ Timers were not started."
  fi

else
  echo "⚠️ Skipping installation. Files remain in ./$SERVICE_DIR"
fi
