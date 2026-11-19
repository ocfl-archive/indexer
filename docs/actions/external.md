# External Actions (`[[Indexer.External]]`)

The `External` action framework allows integrating custom webhooks, microservices, and external format inspection scripts via HTTP endpoints.

---

## Overview

When specialized institutional tools, proprietary analyzers, or external REST microservices (e.g., custom EXIF extractors, specialized audio/video validators, histogram generators) are required, external actions can be registered dynamically in the configuration without altering the Go codebase.

### Key Capabilities
- **Capability Flags**: Configurable per action (`ACTFILE`, `ACTHTTP`, `ACTHTTPS`, `ACTHEAD`, `ACTSTREAM`, etc.).
- **Invocation Types**:
  - `EACTURL`: URL interpolation replacing `[[PATH]]` with the encoded file or target path.
  - `EACTJSONPOST`: Submits structured JSON metadata payload via HTTP POST.
- **MIME Type Gating**: Supports regex filtering on content MIME type before triggering the external endpoint.
- **Default Priority Weight**: `100`

---

## Execution Flow

```mermaid
flowchart TD
    A[Processed File / Target URL] --> B{CanHandle MIME via Regexp?}
    B -->|No Match| C[Skip External Action]
    B -->|Match| D{Check CallType}
    D -->|EACTURL| E[Substitute [[PATH]] placeholder in URL template]
    D -->|EACTJSONPOST| F[Serialize Target Info into JSON Payload]
    E --> G[Dispatch HTTP GET / Request to Remote Endpoint]
    F --> H[Dispatch HTTP POST with JSON to Remote Endpoint]
    G --> I{Inspect HTTP Response}
    H --> I
    I -->|Status 200 OK| J[Parse JSON Result & Merge into ResultV2 Metadata]
    I -->|HTTP Error| K[Record Failure in ResultV2 Errors]
```

---

## Configuration

In `indexer.toml`, define any number of `[[Indexer.External]]` array entries:

```toml
[[Indexer.External]]
name = "validateav"
address = "http://localhost:8083/validateav/[[PATH]]"
calltype = "EACTURL"
mimetype = "^(video|audio)/.*"
ActionCapabilities = ["ACTFILE"]

[[Indexer.External]]
name = "exif"
address = "http://localhost:8083/exif/[[PATH]]"
calltype = "EACTURL"
mimetype = ".*"
ActionCapabilities = ["ACTFILE"]

[[Indexer.External]]
name = "histogram"
address = "http://localhost:8083/histogram/[[PATH]]"
calltype = "EACTURL"
mimetype = "^image/.*"
ActionCapabilities = ["ACTFILE"]
```

### Configuration Fields

| Parameter | Type | Default | Description |
|---|---|---|---|
| `name` | `string` | *(required)* | Unique identifier and metadata key for the external action. |
| `address` | `string` | *(required)* | HTTP URL endpoint. Can include `[[PATH]]` placeholder for `EACTURL`. |
| `calltype` | `string` | `"EACTURL"` | Execution mode: `"EACTURL"` (GET with path substitution) or `"EACTJSONPOST"` (POST JSON payload). |
| `mimetype` | `string` | `".*"` | Regular expression determining which MIME types trigger this action. |
| `ActionCapabilities` | `[]string` | `["ACTFILE"]` | Array of capability tokens (e.g. `["ACTFILE"]`, `["ACTHTTP"]`, `["ACTSTREAM"]`). |

---

## Output Structure

The action populates:

- `Metadata["<name>"]`: Contains the parsed response body returned by the external service.
- `Errors["<name>"]`: Populated if network, timeout, or service-level errors occur.
