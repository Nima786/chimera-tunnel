#!/usr/bin/env python3
import os
import sys
import json
import subprocess
import shutil
import time

# --- Configuration ---
SCRIPT_VERSION = "v1.1-manager" # <-- Updated Version
INSTALL_PATH = '/usr/local/bin/chimera-manager'
CHIMERA_BINARY_PATH = '/usr/local/bin/chimera'
CHIMERA_CONFIG_DIR = '/etc/chimera'
NFT_RULES_FILE = '/etc/nftables.d/chimera-nat.nft'
# --- IMPORTANT: This URL must point to the raw version of THIS script ---
SCRIPT_URL = "https://raw.githubusercontent.com/Nima786/chimera-tunnel/main/chimera-manager.py"

# --- Color Codes ---
class C:
    HEADER = '\033[95m'; BLUE = '\033[94m'; CYAN = '\033[96m'; GREEN = '\033[92m'
    YELLOW = '\033[93m'; RED = '\033[91m'; END = '\033[0m'; BOLD = '\033[1m'

# --- CORRECTED install() function ---
def install():
    print(f"{C.HEADER}--- Starting Chimera Installation ---{C.END}")
    BINARY_URL = "https://github.com/Nima786/chimera-tunnel/releases/download/v0.1.0/chimera"
    temp_binary_path = "/tmp/chimera"
    temp_script_path = "/tmp/chimera-manager.py"

    if os.geteuid() != 0:
        sys.exit(f"{C.RED}Installation requires root privileges. Please run with sudo.{C.END}")

    print(f"{C.CYAN}Installing dependencies (nftables, curl)...{C.END}")
    try:
        subprocess.run(["sudo", "apt-get", "update"], check=True, capture_output=True)
        subprocess.run(["sudo", "apt-get", "install", "-y", "nftables", "curl"], check=True, capture_output=True)
        print(f"{C.GREEN}Dependencies are installed.{C.END}")
    except subprocess.CalledProcessError:
        sys.exit(f"{C.RED}Failed to install dependencies.{C.END}")

    print(f"{C.CYAN}Downloading Chimera core binary...{C.END}")
    try:
        subprocess.run(["curl", "-L", "-o", temp_binary_path, BINARY_URL], check=True, capture_output=True)
        print(f"{C.GREEN}Download complete.{C.END}")
    except subprocess.CalledProcessError as e:
        sys.exit(f"{C.RED}Failed to download binary. Error: {e.stderr.decode()}{C.END}")

    # --- FIX IS HERE: Download the script again instead of copying sys.argv[0] ---
    print(f"{C.CYAN}Downloading the manager script...{C.END}")
    try:
        subprocess.run(["curl", "-L", "-o", temp_script_path, SCRIPT_URL], check=True, capture_output=True)
    except subprocess.CalledProcessError as e:
        sys.exit(f"{C.RED}Failed to download manager script. Error: {e.stderr.decode()}{C.END}")
    # --- END FIX ---

    print(f"{C.CYAN}Installing to /usr/local/bin/...{C.END}")
    try:
        os.chmod(temp_binary_path, 0o755)
        shutil.move(temp_binary_path, CHIMERA_BINARY_PATH)
        
        # Now copy the downloaded script, not the temporary one
        shutil.copy2(temp_script_path, INSTALL_PATH)
        os.chmod(INSTALL_PATH, 0o755)
        os.remove(temp_script_path) # Clean up
        
        print(f"{C.GREEN}Installation successful!{C.END}")
    except Exception as e:
        sys.exit(f"{C.RED}An error occurred during installation: {e}{C.END}")

    print(f"\n{C.BOLD}You can now run the manager with the command:{C.END}")
    print(f"{C.GREEN}sudo chimera-manager{C.END}")

# --- Placeholder Functions ---
def setup_relay_server(): print(f"{C.YELLOW}Not yet implemented.{C.END}")
def generate_client_config(): print(f"{C.YELLOW}Not yet implemented.{C.END}")
def manage_forwarding_rules(): print(f"{C.YELLOW}Not yet implemented.{C.END}")
def uninstall(): print(f"{C.YELLOW}Not yet implemented.{C.END}")

# --- Main Menu Logic ---
def main():
    # This part handles the one-line installer logic
    if not os.path.exists(INSTALL_PATH) and (len(sys.argv) == 1 or sys.argv[1] != '--installed'):
        choice = input(f"{C.HEADER}Install Chimera Tunnel Manager {SCRIPT_VERSION}? (Y/n): {C.END}").lower().strip()
        if choice in ['y', '']:
            install()
        return

    if os.geteuid() != 0:
        sys.exit(f"{C.RED}This script requires root privileges. Please run with sudo.{C.END}")

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
            '1': setup_relay_server, '2': generate_client_config,
            '3': manage_forwarding_rules, '4': uninstall,
            '5': lambda: sys.exit("Exiting.")
        }

        if choice in actions:
            action = actions[choice]
            if action == uninstall:
                action(); break
            else:
                action(); input(f"\n{C.YELLOW}Press Enter to return...{C.END}")
        else:
            print(f"{C.RED}Invalid choice.{C.END}"); time.sleep(1)

if __name__ == '__main__':
    # A small trick to differentiate between the installer run and the installed run
    if len(sys.argv) > 1 and sys.argv[0] == '-c':
         main()
    elif os.path.basename(sys.argv[0]) == os.path.basename(INSTALL_PATH):
        # We are running the installed version, add '--installed' to prevent re-install prompt
        sys.argv.append('--installed')
        main()
    else:
        main()
