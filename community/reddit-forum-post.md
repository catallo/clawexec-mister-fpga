# Reddit / Forum Post — MiSTerClaw Launch

**Target:** r/MiSTerFPGA, r/fpgagaming, misterfpga.org forum

**Title:** 👾🦀 MiSTerClaw — MCP Server for MiSTer FPGA. Let your AI agent control your MiSTer.

**Body:**

Introducing MiSTerClaw, an MCP server that runs on your MiSTer and lets AI assistants control it over the network. It also comes with a CLI client for scripting.

Tell your agent "launch Castlevania on the SNES" and it happens. Ask "what systems do I have?" and it scans your setup and reports back. Ask for a screenshot and it sends one back. Tell it to disable the VGA scaler for a specific core, or to set vsync to a certain value in MiSTer.ini — it just does it.

**Features:**

- Your agent can launch cores and games remotely
- Your agent can search your ROM library across all systems
- Auto-discovers 70+ systems from your ROM folders and installed cores
- Your agent can take screenshots and see what's on screen
- Your agent can check system status and hardware info (CPU, memory, storage, uptime)
- Full shell access — your agent can edit MiSTer.ini settings, tweak core configs, run update scripts, manage files, and fully administer your MiSTer
- Built-in Tailscale VPN integration to easily connect your MiSTer to your network from anywhere
- CLI client included for shell scripts and automation
- Single Go binary, about 3 MB, no dependencies
- Open source (MIT)

**How it works:**

MiSTerClaw speaks MCP (Model Context Protocol), which is the open standard that AI tools use to interact with external systems. It works with OpenClaw, Claude, ChatGPT, Cursor, and any other MCP-compatible client. For tools without MCP support, the CLI client works from the terminal and in scripts.

**Links:**

- GitHub: https://github.com/catallo/misterclaw
- Pre-built binaries: https://github.com/catallo/misterclaw/releases

Happy to hear feedback or answer questions.

---

**Note:** Reddit post used the earlier version (without agent-focused language and shell config examples). Forum post used this final version with BBCode formatting.
