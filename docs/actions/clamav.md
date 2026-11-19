# ClamAV Action (`clamav`)

The `clamav` action performs antivirus scanning and malware detection on ingested files using the open-source ClamAV suite (`clamscan` / `clamdscan`).

---

## Overview

Security screening ensures digital assets are free from known malicious payloads, macros, viruses, and embedded threats prior to preservation. The `clamav` action invokes ClamAV to verify file safety.

### Key Capabilities
- **Capability Flags**: `ACTFILEFULL` (`ACTFILE & ^ACTHEAD` - requires full local file on disk)
- **WSL Support**: Supports Windows host execution forwarding to Linux ClamAV daemons via WSL.
- **Configurable Timeout**: Context-aware process lifecycle termination.
- **Default Priority Weight**: `100`

---

## Execution Flow

```mermaid
flowchart TD
    A[Incoming File on Disk] --> B{Check File Exists and Accessible}
    B -->|No| C[Return File Access Error]
    B -->|Yes| D[Construct clamscan Command with Context Timeout]
    D --> E{Execute ClamAV Process}
    E -->|Exit 0: Clean| F[Record Clean Status in ResultV2]
    E -->|Exit 1: Threat Found| G[Record Virus / Threat Identification]
    E -->|Exit >1: Scan Error| H[Record ClamAV Error]
    F --> I[Finalize ResultV2]
    G --> I
    H --> I
```

---

## Configuration

In `indexer.toml`, configure the `[Indexer.clamav]` section:

```toml
[Indexer.clamav]
enabled = true
clamscan = "/usr/bin/clamdscan" # Path to clamscan or clamdscan
wsl = false                    # Set true if clamscan runs in WSL
timeout = "15s"                # Maximum scan duration
```

### Configuration Fields

| Parameter | Type | Default | Description |
|---|---|---|---|
| `enabled` | `bool` | `true` | Enables or disables antivirus scanning. |
| `clamscan` | `string` | `"/usr/bin/clamdscan"` | Path to the ClamAV scanning binary (e.g., `clamscan` or daemon client `clamdscan`). |
| `wsl` | `bool` | `false` | When `true`, executes ClamAV through `wsl.exe` on Windows hosts. |
| `timeout` | `config.Duration` | `"15s"` | Scan timeout duration. |

---

## Output Structure

The action populates `ResultV2`:

- `Errors["clamav"]`: Populated with threat details if infected, or execution error if scanning fails.
- `Metadata["clamav"]`: Scan summary and verification status.
