# Evolution of code-gehirn 🧠

`code-gehirn` (German for "code brain") was conceived as a lightweight, modular, and privacy-conscious tool for interacting with local knowledge bases (Obsidian vaults, documentation repositories, etc.) using modern LLM capabilities.

## Development Timeline

### Phase 1: The Core Foundation (April 8, 2026 – Morning)
The project was initiated with the goal of building a "code brain" using Go and Qdrant. 
- **Initial Prototype**: The first commit (`feb34e2`) established the core indexing and search functionality using `langchaingo`.
- **Architecture & UI**: Early on, the project structure was formalized, and the basic TUI was introduced (`57a668b`).

### Phase 2: Refinement & Expansion (April 8–9, 2026)
The project expanded beyond the core TUI to support multiple interfaces and improved system observability.
- **Logging & Observability**: A structured logging system was introduced (`502bd9f`, `e679860`), including HTTP traffic interception for debugging RAG operations.
- **CLI Search Enhancements**: Search capabilities were expanded (`b9efa39`) with result formatting and URL extraction utilities.
- **Web UI Introduction**: A browser-based interface was added (`aa9f11c`) with an embedded static frontend, complementing the TUI.
- **Web and TUI Consolidation**: Both interfaces were refined (`cd8476a`, `350bcad`) to share core search and summarization logic.
- **Integration & Merge**: The webui branch was merged into `main` (`3ca92be`), establishing a stable multi-interface architecture.

### Phase 3: Robustness & Local-First Expansion (April 10–12, 2026)
The project focused on reliability, performance, and support for local-only workflows.
- **Ollama Integration**: Support for local LLMs and embedding models via Ollama was added (`8a5124a`), enabling completely private, offline operations.
- **Vector Store Robustness**: Improved handling of embedding dimension mismatches (`b2802fe`) and automated collection isolation (`728390e`, `cc20ddd`) ensured stability across different models.
- **TUI Optimization**: UI lag was reduced through refactoring of asynchronous state management (`0462dff`).
- **CLI Versatility**: A standalone `summarize` command was added (`f35831a`), providing more flexibility for piped workflows and custom prompts.

## Evolution of Interfaces

Each of `code-gehirn`'s three primary interfaces underwent a distinct evolutionary journey, driven by user feedback and the need for greater flexibility.

### 1. CLI Search: From Basic to Specialized
The initial `search` command provided basic proof-of-concept functionality. Subsequent enhancements added practical utilities.
- **Milestone**: The addition of flags like `--urls` and `--all` (`b9efa39`) enabled data extraction patterns—extracting URLs from search results or scanning entire source files.

### 2. TUI: The Interactive Heart
The TUI was built on the `Bubble Tea` framework for real-time interactivity. A critical implementation detail addresses asynchronous timing issues.
- **Milestone**: Request sequencing (`cd8476a`) was implemented to handle cases where multiple searches are in flight. Each search operation is assigned a sequence ID; results are only applied if they match the current active request. This prevents stale results from arriving after a newer search has been initiated, ensuring the UI reflects the most recent query state.

### 3. Web UI: Browser-Based Access
The Web UI provides browser-based access to search and summarization without terminal dependency.
- **Milestone**: Introduced in `aa9f11c`, the web interface includes client-side search result deduplication, error handling, and preview rendering. It shares core search and summary logic with the TUI, enabling feature parity across interfaces.

## Technical Challenges & Lessons

Several implementation challenges emerged during development of the TUI and RAG integration.

### 1. TUI Corruption & Log Redirection
LLM provider SDKs write status messages and warnings to `stderr`, which interferes with Bubble Tea's terminal control in the TUI.
- **Solution**: The `tui` command (`cmd/tui.go`) redirects `os.Stderr` to the `app.log` file during runtime, preventing SDK output from corrupting the terminal interface.

### 2. Handling Terminal Escape Sequences
Some terminal emulators send OSC (Operating System Command) sequences through stdin—e.g., background color queries. These appear as extraneous input characters.
- **Solution**: Input filtering in the search model (`internal/tui/search.go`) restricts input to alphanumeric characters and spaces, naturally rejecting OSC sequences that contain control characters like `;`, `\`, and `]`.

### 3. Asynchronous Search Management
Rapid user input causes multiple searches to be in flight concurrently. Without coordination, a slower search for an earlier query can return results *after* a newer search completes, displaying stale results.
- **Solution**: Request sequencing implemented in the search and summary models. Each search is assigned a sequence ID. Results are only applied if the response ID matches the current `activeReq`. Older searches can be cancelled via `context.Context` if a new search arrives.
- **Implementation Detail**: Uses `tea.Batch()` to send concurrent commands while respecting cancellation semantics, ensuring the UI never displays outdated information.

### 4. Observability vs. UI
Detailed logging for RAG debugging conflicts with a clean terminal interface. A dual-logging approach separates concerns.
- **Solution**: Two log streams created in `internal/logger/`:
    - `app.log`: Application lifecycle events and errors.
    - `api.log`: Raw LLM and vector store request/response bodies captured via custom HTTP transport, useful for debugging prompt interactions and provider behavior.

### 5. Slow Application Startup
Early TUI startup was sequential: embedder → LLM → vector store, leading to delays especially with cloud-based providers.
- **Solution**: Refactored `AppModel.Init()` (`internal/tui/model.go`) to use `tea.Batch()` for concurrent initialization. The embedder and LLM providers initialize in parallel. Vector store connection begins once the embedder is ready, eliminating the LLM initialization dependency.
- **Trade-off**: This removed a safety guarantee (embedder fully ready before vector store connects), but vector store connection is fast enough that this risk is minimal. Parallelization significantly improved perceived startup responsiveness.

### 6. Configuration & Multi-Environment Collisions
Two configuration issues emerged during multi-environment testing:
- **Collection Name Collisions**: Multiple users sharing a single Qdrant instance can overwrite each other's indexes with the default "code-gehirn" collection name.
    - **Solution**: Default collection name now incorporates hostname, OS, and model shortname/shorthash (e.g., `code-gehirn-host-linux-nomic-abcdef12`) for unique isolation.
    - **Lesson**: Environment-specific defaults are essential when software runs in shared systems.
- **Path Resolution**: Configuration files use `~/` for home directory paths, but Viper (the config library) does not expand tildes automatically.
    - **Solution**: Custom path expansion in `internal/config/config.go` manually resolves `~/` before application initialization.
    - **Lesson**: Many Go libraries assume paths are already expanded. Configuration handling must account for platform conventions beyond what the library provides.

### 7. Embedding Model Incompatibility
Switching between embedding models (e.g., OpenAI to Ollama) often causes dimension mismatches in existing vector collections, leading to cryptic server-side errors.
- **Solution**: The `store` package now verifies collection metadata on startup. If a dimension mismatch is detected, it provides a clear error message explaining how to resolve it (either by specifying a new collection or deleting the existing one).
- **Lesson**: Robust RAG systems must proactively validate the compatibility of their embedding models with the stored data to provide a seamless developer experience.

### 8. TUI Blocking & Lag
As search results grew and more data was being processed, the Bubble Tea TUI experienced noticeable lag during long-running operations.
- **Solution**: Asynchronous processing in `internal/tui/` was refactored to offload blocking tasks like result rendering and summarization logic into separate `tea.Cmd` calls, preventing UI main-loop stalls.
- **Lesson**: TUI performance is as sensitive to main-thread blocking as web interfaces. Offloading logic to commands and managing state updates through messages is critical for maintaining responsiveness.

## Core Milestones

### 1. Foundation (Initial Prototype)
The project establishes semantic search using Qdrant (vector database) and Go.

- **Key choice**: [langchaingo](https://github.com/tmc/langchaingo) for LLM orchestration, maintaining provider-agnosticism.
- **Indexer Design**: Respects `.git` boundaries when indexing, processing only markdown content and excluding repository metadata.

### 2. The Terminal Experience (TUI)
An interactive TUI was built using the [Charm Bracelet](https://charm.sh/) ecosystem (Bubble Tea, Lip Gloss, Glamour) to provide a native terminal interface.

- **Real-time Search**: Search results are streamed as the user types.
- **Rich Rendering**: Markdown formatting and code blocks are rendered directly in the terminal.
- **Summarization**: LLM summarization can be triggered on search results without leaving the interface.

### 3. RAG Strategy Evolution: Chunks to Full-Context
Two retrieval approaches are supported:

- **Chunk-Based Retrieval**: Standard vector search returns the top matching 500-token chunks to the LLM.
- **Full-Document Mode**: When `vaultPath` is configured, matching file paths are retrieved from the vector store, and the entire file is read from disk before summarization. This provides more context to the LLM than chunk-only approaches.
- **Path Traversal Protection**: The `vault` package validates all file reads to prevent path traversal attacks and ensure reads stay within the indexed repository (`internal/vault/vault.go`).

### 4. Expansion: Web UI and Multi-Provider Support
Two interface types and multiple LLM providers are now supported:

- **Web UI**: A browser-based interface provides the same search and summarization capabilities as the TUI, built with a Go backend and embedded static frontend.
- **Multi-Provider Support**: Provider layer supports Ollama (local), OpenAI, Anthropic, and Google Gemini, allowing users to choose based on performance and privacy requirements.

### 5. Observability and Debugging
A custom HTTP transport in `internal/logger/` intercepts and logs raw request/response bodies from LLM and vector store calls. This traffic logging is useful for debugging prompt engineering and understanding provider behavior.

### 6. Local-First Power (Ollama Integration)
The addition of Ollama support marks a significant milestone in privacy and cost-efficiency. Users can now run both the LLM and the embedding model locally, keeping all data within their own infrastructure.

### 7. Modular CLI Tools
The introduction of a standalone `summarize` CLI command (`cmd/summarize.go`) allows `code-gehirn` to be used in shell pipelines, separating search from summarization and enabling more complex automation.

## Design Rationale & Key Decisions

### Summarization Strategy: Global vs Per-File
The TUI and Web UI intentionally use the same global summarization strategy: take the original search query and top 5 results, producing a single summary across all matching files rather than summarizing each file individually.
- **Rationale**: Consistency across interfaces and simplicity—users get one coherent answer to their query.
- **Alternative Considered**: Per-file summarization would provide more granular insights but would diverge from the user's original intent and complicate the interface.

### Security Pattern: Path Traversal Protection
The Web UI's `/api/content` endpoint, which serves full file previews, uses `filepath.Rel()` to validate that requested file paths remain within the indexed repository boundaries.
- **Why Not Simple String Matching?**: Go's `filepath.Rel()` properly handles edge cases like `../`, symbolic links, and platform-specific path semantics. Simple `strings.Contains` checks are insufficient for robust path security.
- **Scope**: Ensures all file reads initiated through the web API stay within the vault, preventing malicious path traversal attempts.

### Separation of Concerns: Logging Strategy
The dual-logging approach (app.log for lifecycle, api.log for provider traffic) emerged from a conflict between observability and user experience.
- **Drivers**: LLM provider SDKs write verbose status messages to stderr, corrupting Bubble Tea's terminal rendering. Complete suppression hides useful debugging information.
- **Solution**: Redirect stderr to app.log and separately capture all request/response bodies via custom HTTP transport to api.log. Users get a clean TUI; developers get full debugging context.

## Current State
`code-gehirn` is a functional multi-interface RAG tool with modular architecture supporting both local (Ollama) and cloud (OpenAI, Anthropic, Gemini) LLM providers. As of April 12, 2026, the system is robust against embedding mismatches, features a high-performance TUI, and provides a standalone CLI for summarization tasks.

## Future Directions
Potential areas for future work:
- Support for additional document formats (code files, PDFs, HTML).
- Advanced indexing strategies for very large repositories.
- Additional provider integrations.
- Markdown rendering robustness: The glamour renderer's initialization is timing-sensitive; a more robust fallback strategy would improve reliability in edge cases.
- Configuration hot-reloading (mentioned in early discussions but not implemented).
- Recursive or multi-level summarization for complex queries spanning many documents.
