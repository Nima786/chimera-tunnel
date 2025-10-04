#!/usr/bin/env python3
import os
import sys
import json
import subprocess
import shutil
import time
import re

# --- Configuration ---
SCRIPT_VERSION = "v1.6-final"
INSTALL_PATH = '/usr/local/bin/chimera-manager'
CHIMERA_BINARY_PATH = '/usr/local/bin/chimera'
CHIMERA_CONFIG_DIR = '/etc/chimera'
TUNNELS_DB_FILE = os.path.join(CHIMERA_CONFIG_DIR, 'tunnels.json')
NFT_RULES_FILE = '/etc/nftables.d/chimera-nat.nft'
NFT_NAT_TABLE_NAME = 'chimera_nat'
MAIN_NFT_CONFIG = '/etc/nftables.conf'
SCRIPT_URL = "https://raw.githubusercontent.com/Nima786/chimera-tunnel/main/chimera-manager.py"
RELEASE_BASE_URL = "https://github.com/Nima786/chimera-tunnel/releases/download/v0.3.0"

# --- Color Codes ---
class C:
    HEADER = '\033[95m'; BLUE = '\033[94m'; CYAN = '\033[96m'; GREEN = '\033[92m'
    YELLOW = '\033[93m'; RED = '\033[91m'; END = '\033[0m'; BOLD = '\033[1m'

# --- Helper Functions (unchanged) ---
def clear_screen(): os.system('clear')
def press_enter_to_continue(): input(f"\n{C.YELLOW}Press Enter to return to the menu...{C.END}")
def run_command(command, use_sudo=True, capture=True, text=True, shell=False):
    if use_sudo and os.geteuid() != 0:
        if shell:
            # If shell=True, the command is a string, so prepend 'sudo '
            command = 'sudo ' + command
        else:
            # Otherwise, it's a list
            command = ['sudo'] + command
    try:
        # Pass the shell argument to the underlying subprocess call
        return subprocess.run(command, check=True, capture_output=capture, text=text, shell=shell)
    except subprocess.CalledProcessError:
        return None
def check_and_fix_nftables_config():
    include_line = f'include "/etc/nftables.d/*.nft"'
    if not os.path.exists(MAIN_NFT_CONFIG):
        default_config = f"#!/usr/sbin/nft -f\n\nflush ruleset\n\ntable inet filter {{\n\tchain input {{ type filter hook input priority 0; policy accept; }}\n\tchain forward {{ type filter hook forward priority 0; policy accept; }}\n\tchain output {{ type filter hook output priority 0; policy accept; }}\n}}\n\n{include_line}\n"
        with open(MAIN_NFT_CONFIG, 'w') as f: f.write(default_config)
    with open(MAIN_NFT_CONFIG, 'r') as f:
        if include_line in f.read(): return True
    with open(MAIN_NFT_CONFIG, 'a') as f:
        f.write(f"\n# Added by Chimera Manager\n{include_line}\n")
    with open(MAIN_NFT_CONFIG, 'r') as f:
        if not include_line in f.read():
            print(f"{C.RED}FATAL ERROR: Failed to write to '{MAIN_NFT_CONFIG}'.{C.END}"); return False
    return True
def parse_and_check_ports(ports_str):
    requested_ports = set()
    try:
        for part in ports_str.split(','):
            part = part.strip()
            if '-' in part:
                start, end = map(int, part.split('-'))
                if not (0 < start <= 65535 and 0 < end <= 65535 and start < end): return None, f"Invalid port range '{part}'"
                requested_ports.update(range(start, end + 1))
            else:
                port = int(part)
                if not (0 < port <= 65535): return None, f"Invalid port '{port}'"
                requested_ports.add(port)
    except ValueError:
        return None, f"Invalid format in '{ports_str}'"
    used_ports = set()
    for proto_flag in ['-tlnp', '-ulnp']:
        result = run_command(['ss', proto_flag], capture=True)
        if result:
            for line in result.stdout.splitlines()[1:]:
                match = re.search(r':(\d+)\s', line)
                if match: used_ports.add(int(match.group(1)))
    conflicts = requested_ports.intersection(used_ports)
    if conflicts:
        return None, f"Port(s) already in use: {', '.join(map(str, sorted(conflicts)))}"
    return ", ".join(ports_str.split(',')), None

# --- Core Functions ---
def install():
    print(f"{C.HEADER}--- Starting Chimera Installation ---{C.END}")
    temp_script_path = "/tmp/chimera-manager.py"
    if os.geteuid() != 0: sys.exit(f"{C.RED}Installation requires root privileges.{C.END}")
    print(f"{C.CYAN}Installing dependencies (curl)...{C.END}")
    try:
        subprocess.run(["sudo", "apt-get", "update"], check=True, capture_output=True)
        subprocess.run(["sudo", "apt-get", "install", "-y", "curl", "nftables"], check=True, capture_output=True)
        print(f"{C.GREEN}Dependencies are installed.{C.END}")
    except subprocess.CalledProcessError: sys.exit(f"{C.RED}Failed to install dependencies.{C.END}")
    print(f"{C.CYAN}Downloading the manager script...{C.END}")
    try:
        subprocess.run(["curl", "-L", "-o", temp_script_path, SCRIPT_URL], check=True, capture_output=True)
    except subprocess.CalledProcessError as e: sys.exit(f"{C.RED}Failed to download manager script. Error: {e.stderr.decode()}{C.END}")
    print(f"{C.CYAN}Installing to /usr/local/bin/...{C.END}")
    try:
        shutil.copy2(temp_script_path, INSTALL_PATH); os.chmod(INSTALL_PATH, 0o755); os.remove(temp_script_path)
        print(f"{C.GREEN}Installation successful!{C.END}")
    except Exception as e: sys.exit(f"{C.RED}An error occurred during installation: {e}{C.END}")
    print(f"\n{C.BOLD}You can now run the manager with the command:{C.END}\n{C.GREEN}sudo chimera-manager{C.END}")

def setup_relay_server():
    clear_screen(); print(f"{C.HEADER}--- Setup This Server as a Public Relay ---{C.END}")
    config_path = os.path.join(CHIMERA_CONFIG_DIR, 'server.json')
    if os.path.exists(config_path):
        if input(f"{C.YELLOW}A relay configuration already exists. Overwrite? (y/N): {C.END}").lower().strip() != 'y': return
    if not os.path.exists(CHIMERA_BINARY_PATH):
        print(f"{C.YELLOW}Chimera core binary not found. Downloading it now...{C.END}")
        arch_proc = run_command(['uname', '-m'], use_sudo=False)
        arch = arch_proc.stdout.strip()
        binary_name = ""
        if arch == "x86_64": binary_name = "chimera-amd64"
        elif arch == "aarch64": binary_name = "chimera-arm64"
        else: print(f"{C.RED}Unsupported architecture for relay: {arch}{C.END}"); return
        binary_url = f"{RELEASE_BASE_URL}/{binary_name}"
        try:
            subprocess.run(["curl", "-L", "-o", CHIMERA_BINARY_PATH, binary_url], check=True)
            os.chmod(CHIMERA_BINARY_PATH, 0o755)
        except Exception:
            print(f"{C.RED}Failed to download the Chimera binary for this relay.{C.END}"); return
    print("Select the handshake method this relay will use:")
    print(f"  {C.CYAN}1. Static{C.END} (Simple, requires a public port for handshake)")
    choice = input("Enter choice: ").strip()
    if choice == '1':
        listen_ip = input("Enter the IP address for the relay to listen on (e.g., 0.0.0.0 for all): ").strip() or "0.0.0.0"
        listen_port = input("Enter the port for the relay to listen on (e.g., 8080): ").strip()
        config = { "handshake_method": "static", "listen_address": f"{listen_ip}:{listen_port}" }
        os.makedirs(CHIMERA_CONFIG_DIR, exist_ok=True)
        with open(config_path, 'w') as f: json.dump(config, f, indent=4)
        service_content = f"[Unit]\nDescription=Chimera Relay Server\nAfter=network.target\n\n[Service]\nExecStart={CHIMERA_BINARY_PATH} -config {config_path}\nRestart=always\nUser=root\n\n[Install]\nWantedBy=multi-user.target"
        with open('/etc/systemd/system/chimera-relay.service', 'w') as f: f.write(service_content)
        run_command(['systemctl', 'daemon-reload']); run_command(['systemctl', 'enable', 'chimera-relay.service']); run_command(['systemctl', 'restart', 'chimera-relay.service'])
        print(f"{C.GREEN}Relay server configured and started successfully!{C.END}")
    else:
        print(f"{C.RED}Invalid choice.{C.END}")

def generate_client_config():
    clear_screen(); print(f"{C.HEADER}--- Generate a Client Configuration ---{C.END}")
    print("This will generate a one-line command to set up a client that connects to THIS relay.")
    relay_ip = input("Enter the public IP address of THIS relay server: ").strip()
    relay_port = input("Enter the public port of THIS relay server (e.g., 8080): ").strip()
    config_json = json.dumps({ "handshake_method": "static", "connect_address": f"{relay_ip}:{relay_port}" })
    installer_url = "https://raw.githubusercontent.com/Nima786/chimera-tunnel/main/install-client.sh"
    one_line_command = f"curl -fsSL \"{installer_url}\" | sudo bash -s -- '{config_json}'"
    clear_screen()
    print(f"{C.BOLD}{C.YELLOW}--- ACTION REQUIRED on the Client Server ---{C.END}")
    print("Run the following single, reliable command on the client server:")
    print(f"\n{C.CYAN}{one_line_command}{C.END}\n")

def manage_forwarding_rules():
    while True:
        clear_screen()
        print(f"\n{C.HEADER}--- Manage Port Forwarding Rules ---{C.END}")
        print(f"{C.GREEN}1. Add New Forwarding Rule{C.END}")
        print(f"{C.CYAN}2. List All Forwarding Rules{C.END}")
        print(f"{C.RED}3. Remove Forwarding Rule{C.END}")
        print(f"{C.YELLOW}4. Return to Main Menu{C.END}")
        choice = input("\nEnter your choice: ").strip()
        if choice == '1': add_forwarding_rule()
        elif choice == '2': list_forwarding_rules()
        elif choice == '3': remove_forwarding_rule()
        elif choice == '4': break
        else: print(f"{C.RED}Invalid choice.{C.END}"); time.sleep(1)

def add_forwarding_rule():
    print(f"\n{C.BOLD}--- Add New Forwarding Rule ---{C.END}")
    # For a user-space proxy, the Chimera client itself is the destination.
    # It listens on a local port on the RELAY server.
    # We need to create a new Chimera client service for each port forward.
    # This is a complex logic change. For now, let's assume a simple 1-to-1 mapping.
    print(f"{C.YELLOW}This feature is a work in progress and uses a simplified model.{C.END}")
    public_ports_str = input(f"Enter public port(s) to open on this server (e.g., 5555): ").strip()
    formatted_ports, err = parse_and_check_ports(public_ports_str)
    if err:
        print(f"{C.RED}Error: {err}{C.END}"); return
    
    # In this simple model, the local port on the relay is the same as the public port.
    local_port = public_ports_str.split(',')[0].strip() # Use the first port for the service
    
    dest_ip = input(f"Enter the final destination IP on the client server (e.g., 127.0.0.1): ").strip() or "127.0.0.1"
    dest_port = input(f"Enter the final destination port on the client server (e.g., 5555): ").strip()

    name = f"fwd-{local_port}"
    tunnels = load_tunnels()
    tunnels[name] = {'public_ports': formatted_ports, 'local_port': local_port, 'dest_ip': dest_ip, 'dest_port': dest_port}
    save_tunnels(tunnels)
    generate_and_apply_nft_rules()

def list_forwarding_rules():
    tunnels = load_tunnels()
    if not tunnels: print(f"\n{C.YELLOW}No forwarding rules are configured.{C.END}"); return
    print(f"\n{C.HEADER}--- Configured Forwarding Rules ---{C.END}")
    for name, details in tunnels.items():
        print(f"  {C.CYAN}Public Port(s): {details['public_ports']}{C.END} -> {C.CYAN}{details['dest_ip']}:{details['dest_port']}{C.END}")

def remove_forwarding_rule():
    tunnels = load_tunnels()
    if not tunnels: print(f"\n{C.YELLOW}No rules to remove.{C.END}"); return
    names = list(tunnels.keys())
    print("\n--- Select a Rule to Remove ---")
    for i, name in enumerate(names, 1): print(f"{C.YELLOW}{i}. {tunnels[name]['public_ports']}{C.END}")
    try:
        choice = int(input(f"\nEnter number to remove (0 to cancel): {C.END}"))
        if choice > 0 and choice <= len(names):
            name_to_remove = names[choice - 1]
            del tunnels[name_to_remove]
            save_tunnels(tunnels); generate_and_apply_nft_rules()
            print(f"\n{C.GREEN}Rule for '{name_to_remove}' removed.{C.END}")
    except (ValueError, IndexError): print(f"{C.RED}Invalid selection.{C.END}")

def load_tunnels():
    try:
        with open(TUNNELS_DB_FILE, 'r') as f: return json.load(f)
    except (json.JSONDecodeError, FileNotFoundError): return {}

def save_tunnels(tunnels):
    os.makedirs(CHIMERA_CONFIG_DIR, exist_ok=True)
    with open(TUNNELS_DB_FILE, 'w') as f: json.dump(tunnels, f, indent=4)

# --- REWRITTEN: generate_and_apply_nft_rules is now correct for user-space proxy ---
def generate_and_apply_nft_rules():
    if not check_and_fix_nftables_config(): return
    tunnels = load_tunnels()
    os.makedirs(os.path.dirname(NFT_RULES_FILE), exist_ok=True)
    if not tunnels:
        if os.path.exists(NFT_RULES_FILE): os.remove(NFT_RULES_FILE)
        run_command(['systemctl', 'reload', 'nftables']); return
    
    public_interface = run_command("ip -o -4 route show to default | awk '{print $5}'", shell=True).stdout.strip()
    if not public_interface:
        print(f"{C.RED}Error: Could not determine public interface.{C.END}"); return

    rules = [
        f"table inet {NFT_NAT_TABLE_NAME} {{",
        "\tchain prerouting { type nat hook prerouting priority dstnat; policy accept; }",
        "\tchain postrouting { type nat hook postrouting priority srcnat; policy accept; }",
        "}"
    ]
    for tunnel in tunnels.values():
        # DNAT rule: Redirect public traffic to the local Chimera client port
        rules.append(f'add rule inet {NFT_NAT_TABLE_NAME} prerouting iif "{public_interface}" tcp dport {{ {tunnel["public_ports"]} }} dnat ip to 127.0.0.1:{tunnel["local_port"]}')
        rules.append(f'add rule inet {NFT_NAT_TABLE_NAME} prerouting iif "{public_interface}" udp dport {{ {tunnel["public_ports"]} }} dnat ip to 127.0.0.1:{tunnel["local_port"]}')
    
    # MASQUERADE rule: Fix the return address for all traffic coming from localhost
    rules.append(f'add rule inet {NFT_NAT_TABLE_NAME} postrouting ip saddr 127.0.0.1 oif "{public_interface}" masquerade')
    
    with open(NFT_RULES_FILE, 'w') as f: f.write("\n".join(rules))
    if not run_command(['systemctl', 'reload', 'nftables']):
        run_command(['systemctl', 'restart', 'nftables'])
    print(f"{C.GREEN}nftables rules applied.{C.END}")

def uninstall():
    print(f"{C.YELLOW}This will stop all services and remove all Chimera files.{C.END}")
    if input(f"{C.RED}Are you sure? (y/N): {C.END}").lower().strip() != 'y': return
    
    run_command(['systemctl', 'stop', 'chimera-relay.service'])
    run_command(['systemctl', 'disable', 'chimera-relay.service'])
    
    if os.path.exists('/etc/systemd/system/chimera-relay.service'):
        os.remove('/etc/systemd/system/chimera-relay.service')
    if os.path.exists(INSTALL_PATH): os.remove(INSTALL_PATH)
    if os.path.exists(CHIMERA_BINARY_PATH): os.remove(CHIMERA_BINARY_PATH)
    if os.path.exists(CHIMERA_CONFIG_DIR): shutil.rmtree(CHIMERA_CONFIG_DIR)
    if os.path.exists(NFT_RULES_FILE): os.remove(NFT_RULES_FILE)
    
    run_command(['systemctl', 'daemon-reload'])
    run_command(['systemctl', 'reload', 'nftables'])
    print(f"{C.GREEN}Uninstallation complete.{C.END}")

def main():
    if not os.path.exists(INSTALL_PATH) and (len(sys.argv) == 1 or sys.argv[1] != '--installed'):
        choice = input(f"{C.HEADER}Install Chimera Tunnel Manager {SCRIPT_VERSION}? (Y/n): {C.END}").lower().strip()
        if choice in ['y', '']: install()
        return
    if os.geteuid() != 0: sys.exit(f"{C.RED}This script requires root privileges.{C.END}")
    while True:
        clear_screen()
        print(f"\n{C.HEADER}===== Chimera Tunnel Manager {SCRIPT_VERSION} =====")
        print(f"{C.BLUE}1. Setup This Server as a Public Relay{C.END}")
        print(f"{C.CYAN}2. Generate a Client Configuration{C.END}")
        print(f"{C.GREEN}3. Manage Port Forwarding Rules (on Relay){C.END}")
        print(f"{C.YELLOW}4. Uninstall{C.END}")
        print(f"{C.YELLOW}5. Exit{C.END}")
        choice = input("\nEnter your choice: ").strip()
        actions = {'1': setup_relay_server, '2': generate_client_config, '3': manage_forwarding_rules, '4': uninstall, '5': lambda: sys.exit("Exiting.")}
        if choice in actions:
            action = actions[choice]
            if action == uninstall: action(); break
            else: action(); press_enter_to_continue()
        else: print(f"{C.RED}Invalid choice.{C.END}"); time.sleep(1)

if __name__ == '__main__':
    if len(sys.argv) > 1 and sys.argv[0] == '-c': main()
    elif os.path.exists(INSTALL_PATH) and os.path.basename(sys.argv[0]) == os.path.basename(INSTALL_PATH):
        sys.argv.append('--installed'); main()
    else: main()
