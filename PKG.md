# External Packages and Libraries

This document provides a comprehensive list of all external libraries used by `code-gehirn`, including direct dependencies and significant indirect packages required for its operation.

## CLI and Configuration
The foundation for command-line orchestration and application settings.

- **[spf13/cobra](https://github.com/spf13/cobra)**: CLI framework handling command routing, flags, and help generation.
- **[spf13/viper](https://github.com/spf13/viper)**: Configuration management for YAML, environment variables, and defaults.
- **[fsnotify/fsnotify](https://github.com/fsnotify/fsnotify)**: Cross-platform file system notifications, used for configuration hot-reloading.
- **[pelletier/go-toml/v2](https://github.com/pelletier/go-toml/v2)**: Advanced TOML parser used by the configuration layer.

## Terminal User Interface (TUI) & Rendering
The "Charm" ecosystem and supporting libraries for the interactive terminal experience.

- **[charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)**: The core TUI framework (The Elm Architecture).
- **[charmbracelet/bubbles](https://github.com/charmbracelet/bubbles)**: Reusable components (inputs, lists, viewports).
- **[charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss)**: Styling and layout primitives (colors, borders, alignment).
- **[charmbracelet/glamour](https://github.com/charmbracelet/glamour)**: ANSI-based markdown rendering.
- **[muesli/termenv](https://github.com/muesli/termenv)**: Advanced ANSI color and terminal sequence support.
- **[muesli/reflow](https://github.com/muesli/reflow)**: Text wrapping and reflowing for dynamic terminal resizing.
- **[muesli/ansi](https://github.com/muesli/ansi)** & **[muesli/cancelreader](https://github.com/muesli/cancelreader)**: Low-level terminal I/O and sequence parsing.
- **[mattn/go-runewidth](https://github.com/mattn/go-runewidth)**: Calculation of visual width of characters (crucial for Unicode support).
- **[atotto/clipboard](https://github.com/atotto/clipboard)**: Cross-platform clipboard integration.

## AI and Vector Search Orchestration
The "Brain" layer, primarily powered by the LangChain ecosystem.

- **[tmc/langchaingo](https://github.com/tmc/langchaingo)**: The central SDK for RAG orchestration, provider drivers, and document chains.
- **[Qdrant Go Client](https://github.com/qdrant/go-client)**: Direct and abstracted access to the Qdrant vector database.
- **[pkoukk/tiktoken-go](https://github.com/pkoukk/tiktoken-go)**: BPE tokenization for calculating context limits and cost.
- **[google/generative-ai-go](https://github.com/google/generative-ai-go)**: Specialized SDK for Google Gemini models.
- **[AssemblyAI/assemblyai-go-sdk](https://github.com/AssemblyAI/assemblyai-go-sdk)**: SDK for audio/speech processing (included via LangChainGo).

## Document and Text Processing
Libraries used for parsing, chunking, and highlighting content.

- **[yuin/goldmark](https://github.com/yuin/goldmark)**: High-performance, extensible Markdown parser.
- **[alecthomas/chroma](https://github.com/alecthomas/chroma)**: Syntax highlighting for code blocks within documents.
- **[PuerkitoBio/goquery](https://github.com/PuerkitoBio/goquery)**: jQuery-like HTML parsing for web-based document loaders.
- **[ledongthuc/pdf](https://github.com/ledongthuc/pdf)**: PDF text extraction for non-markdown sources.
- **[microcosm-cc/bluemonday](https://github.com/microcosm-cc/bluemonday)**: HTML sanitizer for safe rendering of untrusted content.
- **[nikolalohinski/gonja](https://github.com/nikolalohinski/gonja)**: Jinja2-compatible templating engine for prompt generation.

## Infrastructure and SDKs
Underlying communication and system-level libraries.

- **[google.golang.org/api](https://pkg.go.dev/google.golang.org/api)**: Core client libraries for Google Cloud/Vertex AI.
- **[google.golang.org/grpc](https://grpc.io/)**: High-performance RPC framework for Qdrant and Cloud service communication.
- **[google.golang.org/protobuf](https://pkg.go.dev/google.golang.org/protobuf)**: Go implementation for Protocol Buffers.
- **[go.opentelemetry.io/otel](https://opentelemetry.io/)**: Observability framework for tracing and metrics.
- **[golang.org/x/net](https://pkg.go.dev/golang.org/x/net)**, **[crypto](https://pkg.go.dev/golang.org/x/crypto)**, **[sys](https://pkg.go.dev/golang.org/x/sys)**: Official Go extension libraries for networking and system primitives.
- **[klauspost/compress](https://github.com/klauspost/compress)**: Optimized compression library for data transfer.
- **[log/slog](https://pkg.go.dev/log/slog)** (Standard Library): Structured logging used across all layers.
- **[net/http](https://pkg.go.dev/net/http)** (Standard Library): Web UI server and REST client backbone.
