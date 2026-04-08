# code-gehirn 🧠

`code-gehirn` (German for "code brain") is a CLI tool for semantic search and summarization of your local markdown knowledge bases (e.g., Obsidian vaults, documentation repositories).

It indexes markdown files from a local git repository into a [Qdrant](https://qdrant.tech/) vector database, enabling semantic search and LLM-powered summarization through a modern terminal interface.

## Features

- **Semantic Search**: Find information based on meaning, not just keywords.
- **LLM Summarization**: Get concise summaries of search results using various LLM providers.
- **Interactive TUI**: A beautiful terminal user interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).
- **Git Integration**: Indexes markdown files directly from your local git repositories.
- **Multi-Provider Support**: Supports multiple embedding and LLM providers (Gemini, Vertex AI, and OpenAI-compatible APIs).

## Prerequisites

- **Go**: 1.26 or higher.
- **Qdrant**: A running instance of Qdrant (local Docker container or Qdrant Cloud).
- **API Keys**: Access to an embedding model and an LLM (e.g., Google Gemini, OpenAI).

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/mfranz/code-gehirn.git
   cd code-gehirn
   ```

2. Build the binary:
   ```bash
   make build
   ```

3. (Optional) Install the binary to `~/bin`:
   ```bash
   make install
   ```

## Configuration

`code-gehirn` looks for a configuration file at `$HOME/.config/code-gehirn/config.yaml` or `./config.yaml`.

Copy the example configuration and fill in your details:
```bash
cp config.yaml.example config.yaml
# Edit config.yaml with your Qdrant and LLM provider details
```

## Usage

### 1. Indexing
Index your markdown files into Qdrant:
```bash
./code-gehirn index /path/to/your/markdown/repo
```

### 2. Semantic Search (CLI)
Perform a quick search from the command line:
```bash
./code-gehirn search "How do I configure the database?"
```

### 3. Interactive TUI
Launch the full interactive experience:
```bash
./code-gehirn tui
```
In the TUI:
- Type to search.
- Use arrow keys or `j`/`k` to navigate results.
- Press `Enter` to select a result and view a summary.
- Press `q` or `Ctrl+C` to quit.

## License

[MIT](LICENSE)
