🚀 Project Chimera: The Unblockable Tunnel
==========================================

Chimera is a next-generation tunneling protocol designed from the ground up to solve the fundamental trade-off between performance and evasion. While traditional protocols like WireGuard are incredibly fast but easy to block, and stealthy protocols like V2Ray/Xray can be complex, Chimera aims to provide the best of both worlds: the raw speed of a minimalist UDP transport with the unblockable, censorship-resistant properties of a protocol that leaves no signature.

✨ Key Features
--------------

*   **⚡ Blazing Fast Performance:** Built on a custom, lightweight UDP transport, Chimera is designed for maximum throughput and minimal latency, making it ideal for high-performance VPNs, game servers, and real-time applications.
*   **👻 Extreme Stealth:** The core data transport has no fixed headers or recognizable patterns. Using strong encryption, its traffic is statistically indistinguishable from random noise, making it a nightmare for DPI firewalls to classify.
*   **🛡️ Unblockable Handshake:** Chimera uses a modular "dead drop" handshake mechanism. By using massive, essential services like Google Cloud Pub/Sub, the initial connection is hidden in a sea of legitimate traffic, bypassing even the most aggressive IP-blocking firewalls.
*   **🌐 Versatile by Design:** The core protocol is a simple, bidirectional data pipe. This allows Chimera to be configured as both a **direct tunnel** (like a personal VPN) and a **reverse tunnel** (for exposing services behind a firewall).
*   **🧩 Modular & Decentralized:** Designed for public use with a "Bring Your Own Keys" model. The project does not rely on any central server, ensuring maximum resilience and security for all users.

🔬 How Chimera Works: A Three-Phase Approach
--------------------------------------------

Chimera intelligently separates the slow, vulnerable parts of a connection from the fast, secure parts.

### Phase 1: The Handshake (The "Secret Introduction")

The most vulnerable part of any tunnel is the initial connection. Chimera makes this invisible.

*   **Static Mode:** A simple, direct IP connection for maximum speed in trusted networks.
*   **Stealth Mode:** Uses a "dead drop" on a massive third-party service (like Google Cloud Pub/Sub). The client and server never connect directly to exchange keys, making the handshake invisible to firewalls that are looking for direct connections.

### Phase 2: The Transport (The "High-Speed Engine")

Once the handshake is complete, Chimera switches to its custom "Scrambled UDP" protocol for data transfer. This protocol is designed for two things: speed and silence. The encrypted packets have no signature and look like random noise on the wire.

### Phase 3: The Decoys (The "Camouflage")

To defeat advanced firewalls that analyze traffic behavior over time, Chimera periodically injects legitimate-looking decoy packets (like DNS queries or WebRTC keep-alives). This breaks the rhythmic pattern of a typical tunnel, further confusing any system trying to classify the traffic.

🎯 Primary Use Cases
--------------------

*   **Censorship Circumvention:** Create a personal VPN that is extremely difficult for state-level firewalls to detect and block.
*   **Resilient Reverse Tunnels:** Expose services from a firewalled network (e.g., a home server) to the public internet, even if the provider is blocking common VPN protocols.

🛠️ Technology Stack
--------------------

*   **Core Engine:** Go (Golang)
*   **Encryption:** NaCl/libsodium (ChaCha20-Poly1305)
*   **Management Script:** Python 3
*   **Firewall Integration:** nftables

🚧 Project Status: In Development 🚧
------------------------------------

Project Chimera is a new and ambitious project. Development will follow a clear, milestone-based roadmap. This is a learning journey, and the goal is to build a truly unique and powerful tool together.

1.  **Milestone 1:** Core Transport Engine (Scrambled UDP)
2.  **Milestone 2:** Static Handshake Implementation
3.  **Milestone 3:** Google Pub/Sub Stealth Handshake
4.  **Milestone 4:** User-Friendly Python Management Script
5.  **Milestone 5:** Decoy Packet Injection System

🤝 How to Contribute
--------------------

This project is just beginning! Contributions, ideas, and feedback are welcome. As the project matures, formal contributing guidelines will be established.
