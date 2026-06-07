
# KVDB.GO - A Thread-Safe, Zero-Allocation LSM Tree

KVDB.GO is a high-throughput, crash-safe key-value storage engine written entirely in pure Go.Designed around the principles of Log-Structured Merge (LSM) Trees, it provides ultra-fast sequential write performance, strict crash durability via a length-prefixed Write-Ahead Log (WAL), and $O(1)$ disk-seek read paths.

This project was engineered to explore the brutal physical realities of modern storage engines—specifically focusing on byte-alignment, OS-level file buffering, memory allocation, and the mechanical sympathy required to bypass Go's garbage collector.


## Architectural Overview

          ┌──────────────┐
            │    Client    │
            │ (HTTP / CLI) │
            └──────┬───────┘
                   │
                   ▼
        ┌──────────────────────┐
        │   HTTP Multiplexer   │
        └─────────┬────────────┘
                   │
                   ▼
        ┌────────────────────────────┐
        │        Storage Node        │
        │                            │
        │           sync.RWMutex     │
        │                │           │
        │       ┌────────┴───────┐   │
        │       ▼                ▼   │
        │ ┌────────────┐   ┌────────────┐
        │ │ Binary WAL │   │  Memtable  │
        │ │  (fsync)   │   │ (Skiplist) │
        │ └────────────┘   └──────┬─────┘
        │                         ▼ Flush
        │                  ┌────────────┐
        │                  │ V2 SSTable │◄── Read (O(1) Seek)
        │                  └────────────┘
        └────────────────────────────┘
## Core Components Breakdown

### 1. Write-Ahead Log (WAL)
To guarantee ACID durability without sacrificing write speed, every mutation is first packed into a zero-allocation, length-prefixed binary array. This array is sequentially appended to the Write-Ahead Log (WAL). The engine enforces a physical fsync syscall to the physical disk platter before acknowledging an OK to the client, ensuring survival against catastrophic power failure.

### 2. Memtable (Concurrent Skiplist)
The in-memory tier utilizes a probabilistic Skiplist. To safely handle thousands of concurrent network clients multiplexed by Go's net/http package, the memory tier is protected by a global sync.RWMutex. This coarse-grained locking architecture allows massive parallel read throughput while safely and strictly serializing the WAL writes.

### 3. V2 Immutable Binary SSTables
When the Memtable exceeds its threshold, it flushes to disk as an immutable `.db` file. The file is not plain text; it is a strictly aligned binary format optimized for $O(1)$ disk seeks. 

The physical file geometry consists of four consecutive blocks:

1. **The Data Blocks:** Tightly packed, sequentially written key-value pairs.
   `[KeyLen uint16 | ValLen uint32 | Key Bytes | Value Bytes]`
2. **The Exact-Offset Index Block:** Appended after the data. Stores the absolute `int64` disk offset for every key.
   `[KeyLen uint16 | Offset uint32 | Key Bytes]`
3. **The 64-bit Bloom Filter Block:** The dynamically sized probabilistic bitset.
   `[m uint64 | k uint64 | bitset uint64 array]`
4. **The 16-Byte Footer:** Fixed-size trailer read on startup to resurrect the system state.
   `[IndexStart uint32 | IndexSize uint32 | BloomStart uint32 | BloomSize uint32]`

Because of this rigid geometry, a `GET` request reads the 16-byte footer, loads the index into RAM, and uses `os.ReadAt` to jump to the exact byte offset of the payload. The read path bypasses OS-level text scanners entirely.

### 4. Bloom Filters
Because LSM trees spread data across multiple files, a read request might require checking
several SSTables on disk. To prevent catastrophic read amplification, each SSTable maintains a
probabilistic Bloom Filter in memory. If the filter returns false, the disk seek is entirely bypassed,
saving immense I/O resources.

### 5. Background Compaction
Deletes in an LSM tree are simply writes with a nil value (known as tombstones). Over time,
tombstones and updated values cause space amplification and degrade read performance. A
background goroutine continuously performs Leveled Compaction, merging overlapping
SSTables, purging obsolete keys, and writing fresh, consolidated tables to disk without locking
the main execution thread.

### 6. Telemetry & Observability
A database without metrics is a ticking time bomb. The engine exposes a real-time /stats endpoint that tracks internal partition boundaries, active memory footprint (keys_in_memory), and disk saturation (sstables_on_disk).

### 7. HTTP API & Interactive CLI
The database is accessible via a robust REST API, backed by Go's native net/http multiplexer
with connection pooling. Furthermore, the repository includes an interactive CLI client
featuring command history, syntax highlighting, and auto-complete for rapid cluster
administration.

## 1. Installation and Usage 

### Prerequisites
```bash
 Go 1.20+
```

### Download dependencies
```bash
go mod download 
```

### Run the server (starts on port 8080)
```bash
go run .
```

## 2. API Usage (cURL)
### Commands

### SET
```bash
curl -X POST http://localhost:8080/set \
     -H "Content-Type: application/json" \
     -d '{"key": "user_1", "value": "Manish"}'
```
### Output
```JSON
{"success":true,"message":"Key Saved Successfully"}
```

### GET
```bash
curl "http://localhost:8080/get?key=user_1"
```

### Output
```JSON
{"success":true,"data":"Manish"}
```

### Delete
```bash
curl -X DELETE "http://localhost:8080/delete?key=user_1"
```
### Output
```bash
{"success":true,"message":"Key Deleted Successfully."}
```

### Stats
```bash
curl http://localhost:8080/stats
```

### Output
```JSON
{"success":true,"data":"{\
"keys_in_memory\":0,\
"partitions\":4,\
"sstables_on_disk\":0
}"}
```

## 3. CLI Usage

```bash
go run cli/main.go cli/cleaner.go
```
### CLI COMMANDS
```
  SET <key> <val>  : Save a new key-value pair
  GET <key>        : Retrieve a value by key
  DELETE <key>     : Remove a key
  PING             : Test server connection
  EXIT             : Close the CLI
  ```

##  Deep Dive: What I Learned (Architectural Tradeoffs)
### Engineering Manifesto: Systems Design and Architectural Tradeoffs
Building a database from scratch removes modern framework layers. It makes an engineer face the tough realities of file systems, memory management, and concurrent design. This system was created by carefully balancing competing factors: memory use versus disk I/O, sequential appending versus random access, and read versus write amplification.

### Here are the main decisions and trade-offs made during development:
- **Storage Engine (LSM Tree vs. B-Tree):** Traditional B-Trees are optimized for reading but face significant Write Amplification and page fragmentation when handling heavy write loads because of the "read-modify-write" cycle. KVDB uses a Log-Structured Merge (LSM) Tree, batching writes in memory and flushing them as immutable SSTables. This change turns slow random disk I/O into fast sequential appends.
- **In-Memory Concurrency (Skiplist vs. B-Tree):** For the in-memory Memtable, we chose a probabilistic Skiplist instead of a B-Tree. Modifying a B-Tree concurrently often requires locking large parts of the tree during node splits. A Skiplist allows for more localized and precise locking, which significantly boosts parallel write rates. Additionally, we used careful pointer management and value semantics to take advantage of Go's escape analysis, keeping short-lived objects on the stack to avoid "stop-the-world" Garbage Collection pauses. 
- **Durability & Torn Writes (WAL):** To withstand power losses without damaging data, all writes are added sequentially to a Write-Ahead Log (WAL) before they are confirmed. To guard against "torn writes" (where a physical disk sector is only partially written during a crash), the WAL ensures strict serialization.
- **Solving Read Amplification (Bloom Filters):** The main downside of an LSM tree is that reading a key might require scanning multiple files. To avoid costly disk seeks for non-existent keys, KVDB creates a Bloom Filter for each SSTable. This mathematically guarantees that if a key is absent, the system can skip unnecessary I/O. If a key is present, a sparse index maps it directly to its disk offset. 
- **The Compaction Tax:** Because LSM trees are append-only, deletes are stored as tombstones. To avoid running out of disk space and to maintain read speeds, a background goroutine performs Leveled Compaction. It continuously merges overlapping SSTables and removes tombstones, carefully balancing the process to prevent stalling the main ingestion thread.

### Summary of Engineering Principles

Building this engine reinforced a fundamental truth of system design: **complexity cannot be destroyed, it can only be moved.**

- If you desire ultra-fast writes, you must move the complexity to the read path (LSM Trees).
- If you desire fast reads on an LSM tree, you must move the complexity to background CPU processing (Compaction & Bloom Filters).
- If you desire high concurrency, you must accept the complexity of data partitioning and fine-grained locking.
