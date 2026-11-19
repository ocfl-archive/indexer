# Indexer Architecture & Design

This document details the internal architecture, lifecycle, streaming execution model, and data aggregation mechanics of the `indexer` engine.

---

## High-Level Overview

`indexer` is a high-performance metadata extraction and format identification engine designed for digital preservation, digital libraries, and archival ingestion workflows (such as OCFL repositories). It runs format identification cascades concurrently over data streams, eliminating redundant I/O passes and maximizing throughput.

```mermaid
graph TB
    subgraph Input ["Data Ingestion"]
        StreamIn["io.Reader (Stream)"]
        FileIn["Local / Remote File"]
    end

    subgraph Core ["Core Engine: ActionDispatcher"]
        MimeDetector["MimeReader / MIME Sniffer"]
        FanOut["io.MultiWriter / Stream Fan-Out"]
        Weighting["MimeRelevance Sorter"]
        Aggregator["ResultV2 Merger"]
    end

    subgraph Actions ["Parallel & Specialized Actions"]
        SF["Siegfried (PRONOM)"]
        YARAX["YARA-X Rule Matcher"]
        XML["XML Schema Parser"]
        JSON["JSON Fieldset Parser"]
        CS["Checksum Calculator"]
        FF["FFmpeg / FFprobe"]
        IM["ImageMagick Identify"]
        TK["Apache Tika (Meta & Fulltext)"]
        AV["ClamAV Scanner"]
        NSRL["NSRL BadgerDB Lookup"]
        EXT["External REST Microservices"]
    end

    subgraph Output ["Standardized Output"]
        Result["ResultV2 Model"]
    end

    StreamIn --> MimeDetector
    FileIn --> MimeDetector
    MimeDetector --> FanOut
    FanOut --> SF
    FanOut --> YARAX
    FanOut --> XML
    FanOut --> JSON
    FanOut --> CS
    FanOut --> FF
    FanOut --> IM
    FanOut --> TK
    FileIn --> AV
    FileIn --> NSRL
    FileIn --> EXT
    SF --> Aggregator
    YARAX --> Aggregator
    XML --> Aggregator
    JSON --> Aggregator
    CS --> Aggregator
    FF --> Aggregator
    IM --> Aggregator
    TK --> Aggregator
    AV --> Aggregator
    NSRL --> Aggregator
    EXT --> Aggregator
    Aggregator --> Weighting
    Weighting --> Result
```

---

## Core Components

### 1. `ActionDispatcher`
The central coordinator (`pkg/indexer/actionDispatcher.go`) responsible for:
- Registering and discovering active actions.
- Action capability checking (`ACTFILE`, `ACTSTREAM`, `ACTHEAD`, etc.).
- Multiplexing a single `io.Reader` across all stream-capable actions concurrently using `io.MultiWriter` and `io.Pipe`.
- Aggregating individual action outcomes into a unified `ResultV2` structure.
- Resolving ambiguous MIME types according to configurable priority weights (`MimeRelevance`).

### 2. `Action` Interface
Every identification tool or analyzer implements the standard `Action` interface (`pkg/indexer/action.go`):

```go
type Action interface {
    Stream(ctx context.Context, contentType string, reader io.Reader, filename string) (*ResultV2, error)
    DoV2(ctx context.Context, filename string) (*ResultV2, error)
    CanHandle(contentType string, filename string) bool
    GetName() string
    GetCaps() ActionCapability
    GetWeight() uint
}
```

### 3. Action Capabilities (`ActionCapability`)
Capabilities indicate execution modes supported by an action:
- `ACTFILE`: Requires a local filepath on disk.
- `ACTSTREAM`: Capable of processing data streamed through `io.Reader`.
- `ACTHEAD`: Only requires the file header / initial bytes.
- `ACTHTTP` / `ACTHTTPS`: Can handle remote HTTP(S) resources directly.

---

## Streaming Execution Lifecycle

When `ActionDispatcher.StreamContext()` is invoked, data flows through a non-blocking parallel pipeline:

```mermaid
sequenceDiagram
    autonumber
    actor Caller
    participant AD as ActionDispatcher
    participant MR as MimeReader
    participant MW as MultiWriter
    participant Actions as Concurrent Actions (Siegfried, Checksum, XML, ...)
    participant Agg as Result Aggregator

    Caller->>AD: StreamContext(ctx, sourceReader, stateFiles, actions)
    AD->>MR: DetectContentType() from initial buffer
    MR-->>AD: Sniffed contentType
    AD->>MW: Create MultiWriter with buffered pipes per action
    par Fan-Out Stream
        AD->>MW: io.Copy(multiWriter, mimeReader)
        MW->>Actions: Concurrent write to action pipes
        Actions->>Actions: Process stream in parallel goroutines
    end
    Actions-->>Agg: Send ResultV2 to results channel
    AD->>Agg: Close pipes & wait for completion (sync.WaitGroup)
    Agg->>Agg: Merge ResultV2 structures
    Agg->>Agg: Sort MIME types via MimeRelevance
    Agg-->>Caller: Final *ResultV2
```

---

## MIME Relevance & Ambiguity Resolution

Different tools might identify different MIME types for the same payload (e.g. `text/plain` vs `application/xml` vs specialized `application/geo+json`). 

`indexer` resolves this deterministically using the configured `MimeRelevance` table:
1. Each MIME type detected by any action is evaluated against `MimeRelevance` regular expressions.
2. Regexes are evaluated with priority given to higher `weight` values, breaking ties with regex pattern specificity.
3. The MIME type with the highest weight becomes `ResultV2.Mimetype`, while all candidates remain recorded in `ResultV2.Mimetypes`.

```toml
[Indexer.MimeRelevance.1]
regexp = "^application/octet-stream$"
weight = 1 # Highly generic, easily down-ranked

[Indexer.MimeRelevance.2]
regexp = "^text/plain$"
weight = 3

[Indexer.MimeRelevance.7]
regexp = "^.+/x-.+"
weight = 80 # Highly specific vendor/experimental format
```

---

## Result Aggregation (`ResultV2`)

`ResultV2` is the standard data model returned by all operations:

```mermaid
classDiagram
    class ResultV2 {
        +map[string]string Errors
        +string Mimetype
        +[]string Mimetypes
        +string Pronom
        +[]string Pronoms
        +map[string]string Checksum
        +uint Width
        +uint Height
        +uint Duration
        +uint64 Size
        +string Type
        +string Subtype
        +map[string]any Metadata
        +Merge(r *ResultV2)
    }
```

- **`Merge()`**: Intelligently combines slice fields (`Mimetypes`, `Pronoms`) with deduplication (`slices.Compact`), takes the maximum dimensions (`Width`, `Height`, `Duration`, `Size`), and merges errors and nested metadata maps without overwriting unrelated data.
- **`GetMetadata[T](r, key)` / `SetMetadata[T](r, key, val)`**: Generic helper functions for type-safe metadata access.
