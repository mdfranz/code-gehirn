# Evolution of code-gehirn 🧠

`code-gehirn` (German for "code brain") was conceived as a lightweight, modular, and privacy-conscious tool for interacting with local knowledge bases (Obsidian vaults, documentation repositories, etc.) using modern LLM capabilities.

## Development Timeline

### Phase 1: The Core Foundation (April 8, 2026)
The project was initiated with the goal of building a "code brain" using Go and Qdrant. 
- **Initial Prototype**: The first commit (`feb34e2`) established the core indexing and search functionality using `langchaingo`.
- **Architecture & UI**: Early on, the project structure was formalized, and the basic TUI was introduced (`57a668b`).
- **RAG Refinement**: Significant improvements were made to the summarization logic and TUI features (`502bd9f`), transitioning from simple searches to a more interactive experience.
- **Logging & Debugging**: A custom HTTP transport was added to the logger to intercept and log LLM traffic, providing critical visibility into the RAG process.

### Phase 2: Refinement & Expansion (April 9, 2026)
The project moved from a terminal-only tool to a multi-interface system.
- **Search Enhancements**: CLI search capabilities were expanded (`b9efa39`), including better result formatting and URL extraction.
- **Web UI Introduction**: A new `web` subcommand was added (`5108ac9`), introducing a browser-based interface powered by an embedded static frontend (`aa9f11c`).
- **Feature Convergence**: The web and TUI interfaces were unified and stabilized (`cd8476a`), ensuring feature parity for search and summarization.
- **Integration & Merge**: The major "webui" feature branch was successfully merged into `main` (`3ca92be`), marking the transition to a stable, multi-platform tool.

## Technical Challenges & Lessons

Building `code-gehirn` revealed several non-obvious challenges in terminal-based RAG applications.

### 1. TUI Corruption & Log Redirection
A major early issue was the TUI "bleeding" or getting corrupted by external logs. Many LLM provider SDKs (e.g., OpenAI, Anthropic) log status messages or warnings directly to `stderr`. In a Bubble Tea application, these logs would write over the interface, leading to visual glitches.
- **Solution**: The `tui` command was updated to globally redirect `os.Stderr` to a dedicated `app.log` file during the TUI session, ensuring the interface remains clean.

### 2. Handling Terminal Escape Sequences
The application encountered "ghost" characters in the search input caused by the terminal itself. Some terminal emulators send OSC (Operating System Command) sequences (like background color queries) as standard input.
- **Solution**: A strict input filter was implemented in the search model, allowing only alphanumeric characters and spaces while explicitly blocking sequences starting with `\033` or containing `;`.

### 3. Asynchronous Search Management
Managing real-time search results proved complex. If a user typed quickly, multiple background searches would be in flight simultaneously. Without careful management, a slower search for an older query could complete *after* a newer search, causing the results to "flicker" back to a stale state.
- **Solution**: A request sequencing and cancellation system was built. Each search is wrapped in a cancellable `context.Context`. When a new search is triggered, any previous search is immediately cancelled, and results are only accepted if they match the current `activeReq` ID.

### 4. Observability vs. UI
Balancing the need for detailed logs (to debug RAG prompts) with a clean user experience required a split-logger architecture.
- **Solution**: Two log streams were created:
    - `app.log`: For application lifecycle events and errors.
    - `api.log`: A dedicated high-volume log that captures the raw request/response bodies of all outbound LLM and vector store calls using a custom HTTP transport.

### 5. Slow Application Startup
Initial versions of the TUI suffered from a sequential "waterfall" startup. The application would first initialize the embedder, then wait for the LLM, and finally connect to the vector store. This led to a sluggish user experience, especially when using cloud-based providers.
- **Solution**: The `AppModel.Init()` was refactored to use `tea.Batch` for parallel initialization. The embedder and LLM providers are now initialized concurrently. The vector store connection starts as soon as the embedder is ready, without waiting for the LLM, cutting the total startup time by nearly 50%.

### 6. Configuration & Multi-Environment Collisions
As the tool was tested across different machines, two major configuration hurdles emerged:
- **Collection Name Collisions**: Multiple users or environments sharing a single Qdrant instance would overwrite each other's indexes if they used the default "code-gehirn" collection name. 
    - **Solution**: The default collection name was changed to be globally unique by incorporating the local hostname and OS (e.g., `code-gehirn-my-laptop-linux`).
- **Path Resolution**: Users expected `~/` to work in configuration paths for log files, but the underlying configuration library (Viper) does not automatically expand tildes in strings.
    - **Solution**: A custom path expansion utility was added to the configuration loader to manually resolve home directory prefixes before the application starts.

## Core Milestones

### 1. Foundation (Initial Prototype)
The project started with a focus on **Semantic Search**. By leveraging the Qdrant vector database and Go, the goal was to provide a faster and more meaningful search experience than simple keyword-based `grep`. 

- **Key choice**: Use [langchaingo](https://github.com/tmc/langchaingo) for LLM orchestration to maintain provider-agnosticism from day one.
- **Git Integration**: The indexer was designed to respect `.git` boundaries, ensuring only relevant markdown content is indexed while skipping internal repository metadata.

### 2. The Terminal Experience (TUI)
Recognizing that developers live in the terminal, a robust **Interactive TUI** was built using the [Charm Bracelet](https://charm.sh/) ecosystem (Bubble Tea, Lip Gloss, Glamour). 

- **Real-time Search**: Instant feedback as the user types queries.
- **Rich Rendering**: Markdown previews directly in the terminal, preserving formatting and code blocks.
- **In-place Summarization**: The ability to trigger an LLM summary of search results without leaving the interface.

### 3. RAG Strategy Evolution: Chunks to Full-Context
The retrieval-augmented generation (RAG) strategy evolved significantly to improve summary quality:

- **Initial Approach**: Standard chunk-based retrieval, where the LLM only saw the 500-token snippets that matched the search query.
- **Enhanced "Full-Document" Mode**: Introduced a hybrid model. If a `vaultPath` is configured, the system retrieves matching file paths from the vector store but then reads the **entire document** from the local filesystem. This provides the LLM with the full context of the relevant files, leading to much richer and more accurate summaries.
- **Security (The Vault)**: To support full-document reads safely, a `vault` package was introduced to prevent path traversal attacks and ensure all reads are constrained within the indexed repository.

### 4. Expansion: Web UI and Multi-Provider Support
To make the "code brain" accessible beyond the terminal:

- **Web UI**: An experimental browser-based interface was added, featuring the same search and summarization capabilities as the TUI, built with a Go backend and an embedded static frontend.
- **Broad Provider Support**: The provider layer was abstracted to support **Ollama** (for 100% local operation), **OpenAI**, **Anthropic**, and **Google Gemini**, allowing users to choose their balance of performance and privacy.

### 5. Observability and Debugging
A custom HTTP transport was implemented in the `logger` package to intercept and log the raw request/response bodies of all LLM and vector store calls. This "traffic interception" became a vital tool for debugging prompt engineering and understanding how the LLM interprets the provided context.

## Current State and Future
Today, `code-gehirn` is a stable tool for local RAG. The architecture is modular, allowing for easy addition of new providers or UI frontends. 

**Future areas of exploration include:**
- Expanding beyond Markdown to other text-based formats (code files, PDFs).
- Stabilizing the Web UI for a production-ready experience.
- Implementing advanced indexing techniques like recursive summarization for massive repositories.
