# crusty-buffer

# What is an **Article** Is in crusty-buffer

An **Article** is the atomic unit of truth in this system. Everything else exists to move it, mutate it, or store it without screwing it up. Below is what that means, depending on who’s looking.

---

## 1. The User’s View

To the user, an **Article** is a **frozen webpage**, not a bookmark.

* **Time-Travel, Not a Pointer**
  A bookmark is lazy. It just remembers a URL and hopes the internet keeps its promises.
  An Article is a hard capture of the page at a specific moment. Run `crusty add`, and that version is locked in. If the site mutates, rots, or vanishes, the Article doesn’t care.

* **Clean by Force**
  This is not raw HTML. Ads, popups, cookie nags, tracking junk—gone. What’s left is readable content and essential images. Think Reader Mode, but permanent and local.

* **Offline, Actually Offline**
  No sync dependency. No cloud safety net. The Article lives on disk. Plane mode still works. If that scares you, good—you now understand the tradeoff.

---

## 2. for the one reading this repo

To someone reading the codebase, the `Article` struct in `internal/model/article.go` is the **core domain entity**. Not a DTO. Not a helper. The spine of the system.

* **Identity Is Not the URL**
  The primary key is `uuid.UUID`, not the URL. That’s not accidental.
  Same URL saved twice equals two Articles. That’s versioning, whether you like the word or not.

* **It Has States Because Work Happens Later**
  `pending`, `archived`, `failed` means one thing: this system is asynchronous.
  The CLI does not “save articles.” It schedules work and walks away.

* **Designed to Cross Boundaries**
  JSON tags are not decoration. This struct is serialized, shoved into Redis, passed through queues, and rehydrated elsewhere.
  If you don’t think about that, you will break it.

---

## 3. Developer View

The Article is a **dangerous payload** moving across layers that have very different tolerance for size and stupidity.

* **One Concept, Two Physical Forms**
  The Article is logically one thing, but physically split:

  * **Lightweight Metadata**
    ID, URL, Title, Status. This fits in Redis without lighting RAM on fire.
  * **Heavyweight Content**
    The cleaned body. This does *not* belong in Redis. Ever. It lives in BadgerDB, where blobs are allowed to exist without shame.

* **The Contract Between Systems**
  This struct is the handshake between the CLI and the worker.

  * CLI creates a **shell**: ID + URL, `Status: pending`
  * Worker fills it in: Title + Content, `Status: archived`

  Break this contract and the pipeline collapses.

* **The Source of Pain**
  This is where bugs breed.
  The recent one was classic: `json.Marshal` happily tried to serialize the entire article body and jam it into Redis.
  The computer did exactly what it was told. We were the idiots.

---

## Mental Model

```text
        USER sees:             REPO VIEWER sees:           WE see:
   +-------------------+    +---------------------+    +------------------+
   |  Saved Snapshot   |    |  Core Domain Entity |    |  Volatile Payload|
   |                   |    |  (Struct)           |    |                  |
   +--------+----------+    +----------+----------+    +--------+---------+
            |                          |                        |
   - Offline forever          - UUID identity           - Split storage
   - No ads                   - State machine           - Redis vs Badger
   - Immutable                - Async workflow          - Serialization traps
```
---

### work-flowchart
```text
     [ PRODUCER ]                     [ QUEUE ]                    [ CONSUMER ]
   (The CLI Command)                (Redis List)                   (The Worker)

   1. "crusty add URL"             2. Waiting Room               3. Processing Lab
   +-----------------+            +---------------+            +------------------+
   |                 |            |               |            |                  |
   |  Create Article |   Push     | [ ID: abc ]   |    Pop     | 1. Fetch HTML    |
   |  (Status:       +----------> | [ ID: xyz ]   +--------->  | 2. Clean HTML    |
   |    PENDING)     |            |               |            | 3. Compress      |
   |                 |            +---------------+            |                  |
   +-----------------+                                         +--------+---------+
           |                                                            |
           v                                                            v
    (Metadata Saved)                                             (Content Saved)
    +--------------+                                           +------------------+
    | REDIS HASH   | <----------------------------------------+| BADGER DB        |
    | "Title: ..." |           (Update Status: ARCHIVED)       | "<html>...</html>|
    | "Status: ..."|                                           +------------------+
    +--------------+
```
--- 

## The Three States of an Article
The Worker is effectively a state machine manager. An article moves through three distinct phases:

- Pending (The Promise):

  - Where: Only in Redis

  - Data: We have the URL and an ID. We have no title (yet) and no content

  - User sees: "Processing..." in the UI

- Processing (The Work):

  - Where: In the Worker's memory

  - Action: The worker is currently downloading the page and stripping out ads using go-readability

- Archived (The Result):

  - Where: Metadata in Redis, Content in Badger

  - Data: We now have the real Title, the Excerpt, and the full HTML

  - User sees: The final article in the list

---

## Verification

### Prerequisites

You **must** have Redis running locally

```bash
docker run -d -p 6379:6379 redis
```

### Run

```bash
go mod init crusty-buffer && go mod tidy
```

### Build

```bash
make build
```

---

### Start the Server **(Terminal 1)**

This starts the **Consumer (Worker)**
It will block and wait for jobs

```bash
./bin/crusty server
```

Expected output:

```text
INFO Server running. Press Ctrl+C to stop.
INFO Worker started. Waiting for jobs...
```

To shut it down cleanly, press:

```
q + Enter 
or 
Ctrl+C
```

---

### Add a URL **(Terminal 2)**

This acts as the **Producer**.

```bash
./bin/crusty add https://example.com
```

Expected output:

```text
INFO Article queued {"id": "...", "url": "https://example.com"}
```

Now look back at **Terminal 1**.
The worker should wake up immediately:

```text
INFO Processing started {"job_id": "..."}
INFO Downloading {"url": "https://example.com"}
INFO Archiving complete {"title": "Example Domain"}
```

If you don’t see this, something is broken. Fix it.

---

### Step 4: Verify Persistence

Stop the server (`q + Enter or Ctrl+C`) and start it again.

The data is **not lost**.
It’s persisted in **Badger / Redis**.

There’s no `list` command yet, but the files in:

```text
./badger-data
```
