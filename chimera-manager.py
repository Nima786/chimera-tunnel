#!/usr/bin/env python3
import os
import sys
import json
import subprocess
import shutil
import time

# --- Configuration ---
SCRIPT_VERSION = "v1.0-manager"
INSTALL_PATH = '/usr/local/bin/chimera-manager'
CHIMERA_BINARY_PATH = '/usr/local/bin/chimera'
CHIMERA_CONFIG_DIR = '/etc/chimera'
NFT_RULES_FILE = '/etc/nftables.d/chimera-nat.nft'

# --- Color Codes ---
class C:
    HEADER = '\033[95m'; BLUE = '\033[94m'; CYAN = '\033[96m'; GREEN = '\032[92m'
    YELLOW = '\033[93m'; RED = '\033[91m'; END = '\033[0m'; BOLD = '\033[1m'

# --- Placeholder Functions (We will implement these one by one) ---
def install():
    """
    Installs the Chimera manager and the core binary.
    """
    print(f"{C.HEADER}--- Starting Chimera Installation ---{C.END}")

    # --- 1. Define URLs and Paths ---
    # IMPORTANT: Replace this with the real URL you copied from your GitHub Release!
    BINARY_URL = "https://github.com/Nima786/chimera-tunnel/releases/download/v0.1.0/chimera"
    
    # We will use a temporary file for the download
    temp_binary_path = "/tmp/chimera"

    # --- 2. Check for root privileges ---
    if os.geteuid() != 0:
        sys.exit(f"{C.RED}Installation requires root privileges. Please run with sudo.{C.END}")

    # --- 3. Install Dependencies ---
    print(f"{C.CYAN}Checking and installing dependencies (nftables, curl)...{C.END}")
    try:
        subprocess.run(["sudo", "apt-get", "update"], check=True)
        subprocess.run(["sudo", "apt-get", "install", "-y", "nftables", "curl"], check=True)
        print(f"{C.GREEN}Dependencies are installed.{C.END}")
    except subprocess.CalledProcessError:
        sys.exit(f"{C.RED}Failed to install dependencies. Please install them manually and try again.{C.END}")

    # --- 4. Download the Chimera Binary ---
    print(f"{C.CYAN}Downloading the Chimera core binary from GitHub...{C.END}")
    try:
        # Use curl to download the file. The -L flag follows redirects.
        subprocess.run(
            ["curl", "-L", "-o", temp_binary_path, BINARY_URL],
            check=True, capture_output=True
        )
        print(f"{C.GREEN}Download complete.{C.END}")
    except subprocess.CalledProcessError as e:
        sys.exit(f"{C.RED}Failed to download the Chimera binary. Error: {e.stderr.decode()}{C.END}")

    # --- 5. Install the Binary and Manager Script ---
    print(f"{C.CYAN}Installing Chimera to /usr/local/bin/...{C.END}")
    try:
        # Make the downloaded binary executable
        os.chmod(temp_binary_path, 0o755)
        # Move it to the system path
        shutil.move(temp_binary_path, CHIMERA_BINARY_PATH)

        # Copy the manager script itself to the system path
        shutil.copy2(sys.argv[0], INSTALL_PATH)
        os.chmod(INSTALL_PATH, 0o755)
        
        print(f"{C.GREEN}Installation successful!{C.END}")
    except Exception as e:
        sys.exit(f"{C.RED}An error occurred during installation: {e}{C.END}")

    print(f"\n{C.BOLD}You can now run the manager from anywhere with the command:{C.END}")
    print(f"{C.GREEN}sudo chimera-manager{C.END}")

def setup_relay_server():
    print(f"{C.YELLOW}Setup Relay Server function not yet implemented.{C.END}")
    # TODO: Wizard to create server config.json
    # TODO: Create and start systemd service for 'chimera listen'

def generate_client_config():
    print(f"{C.YELLOW}Generate Client Config function not yet implemented.{C.END}")
    # TODO: Wizard to create client config.json
    # TODO: Display the config for the user to copy

def manage_forwarding_rules():
    print(f"{C.YELLOW}Manage Forwarding Rules function not yet implemented.{C.END}")
    # TODO: Sub-menu to add/remove nftables rules

def uninstall():
    print(f"{C.YELLOW}Uninstall function not yet implemented.{C.END}")
    # TODO: Stop services, remove all created files

# --- Main Menu Logic ---
def main():
    if os.geteuid() != 0:
        sys.exit(f"{C.RED}This script requires root privileges. Please run with sudo.{C.END}")

    if not os.path.exists(INSTALL_PATH):
        choice = input(f"{C.HEADER}Install Chimera Tunnel Manager {SCRIPT_VERSION}? (Y/n): {C.END}").lower().strip()
        if choice in ['y', '']:
            install()
        return

    while True:
        os.system('clear')
        print(f"\n{C.HEADER}===== Chimera Tunnel Manager {SCRIPT_VERSION} =====")
        print(f"{C.BLUE}1. Setup This Server as a Public Relay{C.END}")
        print(f"{C.CYAN}2. Generate a Client Configuration{C.END}")
        print(f"{C.GREEN}3. Manage Port Forwarding Rules (on Relay){C.END}")
        print(f"{C.YELLOW}4. Uninstall{C.END}")
        print(f"{C.YELLOW}5. Exit{C.END}")
        choice = input("\nEnter your choice: ").strip()

        actions = {
            '1': setup_relay_server,
            '2': generate_client_config,
            '3': manage_forwarding_rules,
            '4': uninstall,
            '5': lambda: sys.exit("Exiting.")
        }

        if choice in actions:
            action = actions[choice]
            if action == uninstall:
                action()
                break
            else:
                action()
                input(f"\n{C.YELLOW}Press Enter to return to the menu...{C.END}")
        else:
            print(f"{C.RED}Invalid choice.{C.END}")
            time.sleep(1)

if __name__ == '__main__':
    main()
