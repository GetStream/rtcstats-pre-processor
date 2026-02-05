# WebRTC RTCStatsReport Compression Spec (LLM-friendly)

**Goal:** Compress raw `getstats` RTCStatsReport data into a token-optimized format while preserving all diagnostically meaningful metrics. Counters are converted to deltas; gauges are kept as-is.

**Input format:** `["getstats", scope, {id: {field: value, ...}, ...}, ts]`

**Scopes observed:**
- `"0-pub"` — publisher peer connection stats
- `"0-sub"` — subscriber peer connection stats
- `"1-sfu-dpk-..."` — SFU-level connection quality (CQ only)

---

## 1) Report Type Classification

Raw getstats entries lack explicit `type` fields on most entries. Classify each entry by its **field fingerprint**:

| Report Type | Fingerprint | Compressed Key |
|---|---|---|
| Outbound RTP Video | has `framesEncoded` + `bytesSent` | `out_v` |
| Outbound RTP Audio | has `bytesSent` + `headerBytesSent`, no `framesEncoded` | `out_a` |
| Inbound RTP Audio | has `bytesReceived` + (`totalAudioEnergy` or `audioLevel`) | `in_a` |
| Inbound RTP Video | has `bytesReceived` + `framesDecoded` | `in_v` |
| Remote-Inbound RTP | has `roundTripTime` or `roundTripTimeMeasurements`, no `responsesReceived`/`currentRoundTripTime` | `rtt` |
| Candidate Pair (active) | has `responsesReceived` or `currentRoundTripTime` | `cp` |
| Candidate Pair (relay/check) | has `bytesSent` + `remoteTimestamp`, no `responsesReceived` | `cp_r` |
| Media Source (video) | ID matches `mediasource_video_*` and has `frames` | `ms` |
| Media Source (audio) | ID matches `mediasource_audio_*` — always timestamp-only | DROP |
| Connection Quality | key is literally `"CQ"` | `cq` |
| Timestamp-only | only field is `timestamp` (no other fields) | DROP |

### Classification priority

Apply rules **top-to-bottom** (first match wins). This avoids ambiguity when fields overlap between candidate-pair and outbound entries.

---

## 2) Counter vs Gauge Handling

### Counters (`Δ`)

Monotonically increasing values that accumulate over the session lifetime. Store the **delta** from the previous sample for the same SSRC/entry ID.

- First sample for a given ID: store the absolute value.
- Subsequent samples: store `current - previous`.
- If delta is `0`, omit the field entirely.

### Gauges (`G`)

Point-in-time snapshot values. Store the **latest value** as-is.

- If value is `0`, `null`, or absent: omit the field.

### Sparse fields

Fields that appear only sometimes (e.g., `discardedPackets`, `silentConcealedSamples`, `freezeCount`): **include when present and non-zero, omit when zero or absent**.

---

## 3) Per-Report-Type Specs

### 3.1) Outbound RTP Video — `out_v`

Scope: `0-pub`. Multiple entries per sample (one per simulcast layer).

| Original Field | Short Key | Type | Notes |
|---|---|---|---|
| `bytesSent` | `bs` | Δ | |
| `headerBytesSent` | `hbs` | Δ | |
| `packetsSent` | `ps` | Δ | |
| `framesEncoded` | `fe` | Δ | |
| `framesPerSecond` | `fps` | G | absent on some samples |
| `qpSum` | `qp` | Δ | |
| `totalEncodeTime` | `tet` | Δ | seconds; Δ gives per-interval encode time |
| `totalEncodedBytesTarget` | `tebt` | Δ | target budget delta |
| `pliCount` | `pli` | Δ | sparse — include when non-zero |
| `hugeFramesSent` | `hfs` | Δ | sparse — include when non-zero |

**Drop:**
- `framesSent` — always equals `framesEncoded` in observed data
- `frameHeight`, `frameWidth` — logged once in track setup, rarely changes
- `remoteId` — internal WebRTC correlation ID
- `timestamp` — redundant with envelope `ts`

**Example compressed output (single layer):**
```json
{"bs":1901627,"hbs":35460,"ps":1773,"fe":298,"fps":30,"qp":5443,"tet":1.613,"tebt":9826634}
```

---

### 3.2) Outbound RTP Audio — `out_a`

Scope: `0-pub`. Single entry per sample.

| Original Field | Short Key | Type | Notes |
|---|---|---|---|
| `bytesSent` | `bs` | Δ | |
| `headerBytesSent` | `hbs` | Δ | |
| `packetsSent` | `ps` | Δ | |

**Drop:**
- `timestamp`

**Example compressed output:**
```json
{"bs":24504,"hbs":3440,"ps":172}
```

---

### 3.3) Inbound RTP Audio — `in_a`

Scope: `0-sub`. Single entry per sample.

| Original Field | Short Key | Type | Notes |
|---|---|---|---|
| `bytesReceived` | `br` | Δ | |
| `headerBytesReceived` | `hbr` | Δ | |
| `packetsReceived` | `pr` | Δ | |
| `jitter` | `j` | G | seconds |
| `audioLevel` | `al` | G | 0.0–1.0; current audio level |
| `totalAudioEnergy` | `tae` | Δ | |
| `totalSamplesDuration` | `tsd` | Δ | seconds |
| `totalSamplesReceived` | `tsr` | Δ | |
| `concealedSamples` | `cs` | Δ | sparse |
| `concealmentEvents` | `ce` | Δ | sparse |
| `removedSamplesForAcceleration` | `rsa` | Δ | sparse |
| `silentConcealedSamples` | `scs` | Δ | sparse — very rare |
| `jitterBufferDelay` | `jbd` | Δ | ms total; Δ gives per-interval |
| `jitterBufferEmittedCount` | `jbe` | Δ | |
| `jitterBufferMinimumDelay` | `jbm` | Δ | |
| `jitterBufferTargetDelay` | `jbt` | Δ | |

**Drop:**
- `lastPacketReceivedTimestamp` — redundant with envelope `ts`
- `timestamp`

**Example compressed output:**
```json
{"br":43873,"hbr":3588,"pr":299,"j":0.005,"al":0.076,"tae":0.445,"tsd":10.0,"tsr":480960,"cs":600,"ce":1,"jbd":20928,"jbe":282240,"jbm":14976,"jbt":14976}
```

---

### 3.4) Inbound RTP Video — `in_v`

Scope: `0-sub`. Single entry per sample.

| Original Field | Short Key | Type | Notes |
|---|---|---|---|
| `bytesReceived` | `br` | Δ | |
| `headerBytesReceived` | `hbr` | Δ | |
| `packetsReceived` | `pr` | Δ | |
| `jitter` | `j` | G | seconds |
| `framesDecoded` | `fd` | Δ | |
| `framesReceived` | `fr` | Δ | keep — can diverge from `fd` under loss |
| `framesPerSecond` | `fps` | G | absent on some samples |
| `framesAssembledFromMultiplePackets` | `fam` | Δ | |
| `qpSum` | `qp` | Δ | |
| `totalDecodeTime` | `tdt` | Δ | seconds |
| `totalInterFrameDelay` | `tifd` | Δ | seconds |
| `totalSquaredInterFrameDelay` | `tsid` | Δ | for jitter variance |
| `totalAssemblyTime` | `tat` | Δ | seconds |
| `totalProcessingDelay` | `tpd` | Δ | seconds |
| `jitterBufferDelay` | `jbd` | Δ | |
| `jitterBufferEmittedCount` | `jbe` | Δ | |
| `jitterBufferMinimumDelay` | `jbm` | Δ | |
| `jitterBufferTargetDelay` | `jbt` | Δ | |
| `packetsLost` | `pl` | Δ | sparse |
| `packetsDiscarded` | `pd` | Δ | sparse (same as `discardedPackets`) |
| `discardedPackets` | — | — | alias for `packetsDiscarded`; prefer `pd` |
| `nackCount` | `nk` | Δ | sparse |
| `keyFramesDecoded` | `kfd` | Δ | sparse |
| `freezeCount` | `fzc` | Δ | sparse |
| `totalFreezesDuration` | `fzd` | Δ | sparse; seconds |
| `framesDropped` | `fdr` | Δ | sparse |

**Drop:**
- `lastPacketReceivedTimestamp`
- `timestamp`

**Example compressed output:**
```json
{"br":1321902,"hbr":28824,"pr":1201,"j":0.015,"fd":149,"fr":149,"fps":14,"fam":149,"qp":21422,"tdt":0.266,"tifd":9.97,"tsid":0.676,"tat":1.898,"tpd":15.007,"jbd":14.179,"jbe":150,"jbm":11.07,"jbt":11.07}
```

---

### 3.5) Remote-Inbound RTP — `rtt`

Scope: `0-pub`. Multiple entries (one per outbound SSRC that has remote feedback).

| Original Field | Short Key | Type | Notes |
|---|---|---|---|
| `roundTripTime` | `rtt` | G | seconds; latest measurement |
| `jitter` | `j` | G | seconds |
| `packetsReceived` | `pr` | Δ | packets received by remote |
| `totalRoundTripTime` | `trtt` | Δ | |
| `roundTripTimeMeasurements` | `rttm` | Δ | |

**Drop:**
- `type`, `kind`, `mediaType` — already classified
- `codecId`, `ssrc`, `localId` — internal WebRTC correlation IDs
- `fractionLost`, `packetsLost` — available on "full" variant only, rarely present
- `timestamp`

**Note:** Some samples include a "full" variant with `type: "remote-inbound-rtp"` plus extra fields (`codecId`, `kind`, `ssrc`, etc.). These are classified identically and the extra fields are dropped.

**Example compressed output:**
```json
{"rtt":0.013,"j":0.0009,"pr":680,"trtt":0.130,"rttm":10}
```

---

### 3.6) Candidate Pair — `cp`

Two sub-types are merged into one compressed output:

#### Active Candidate Pair
Scope: `0-pub` or `0-sub`. One entry per sample per peer connection.

| Original Field | Short Key | Type | Notes |
|---|---|---|---|
| `bytesSent` | `bs` | Δ | |
| `bytesReceived` | `br` | Δ | |
| `currentRoundTripTime` | `rtt` | G | seconds; absent on some samples |
| `responsesReceived` | `rr` | Δ | STUN responses |
| `totalRoundTripTime` | `trtt` | Δ | |

**Drop:**
- `lastPacketReceivedTimestamp`, `lastPacketSentTimestamp` — redundant with envelope `ts`
- `timestamp`

**Example (active):**
```json
{"bs":2849206,"br":10444,"rtt":0.012,"rr":2,"trtt":0.025}
```

#### Relay/Check Candidate Pair (`cp_r`)
Scope: `0-sub`. Entries with `remoteTimestamp` but no `responsesReceived`/`currentRoundTripTime`.

| Original Field | Short Key | Type | Notes |
|---|---|---|---|
| `bytesSent` | `bs` | Δ | |
| `packetsSent` | `ps` | Δ | |
| `remoteTimestamp` | `rts` | G | remote clock; keep latest |

**Drop:**
- `timestamp`

**Example (relay):**
```json
{"bs":42279,"ps":276,"rts":1770106214224.49}
```

---

### 3.7) Connection Quality — `cq`

Scope: SFU scope only (e.g., `"1-sfu-dpk-..."` ). Key is literally `"CQ"`.

| Original Field | Short Key | Type | Notes |
|---|---|---|---|
| `score` | `s` | G | 0–100 current score |
| `avgScore` | `as` | G | running average |
| `mosScore` | `mos` | G | MOS (1.0–5.0); sparse |

All fields are gauges. Include only fields present and non-zero in each sample.

**Drop:**
- `timestamp`

**Example compressed output:**
```json
{"s":94.18,"as":98.6,"mos":4.43}
```

---

### 3.8) Media Source (video) — `ms`

Scope: `0-pub`. ID matches `mediasource_video_*`.

| Original Field | Short Key | Type | Notes |
|---|---|---|---|
| `frames` | `f` | Δ | total frames produced by source |
| `framesPerSecond` | `fps` | G | |

**Drop:**
- `timestamp`
- Media source audio entries — always timestamp-only in observed data; drop entirely.

**Example compressed output:**
```json
{"f":298,"fps":30}
```

**Note:** Media source video largely duplicates outbound RTP `framesEncoded`. Consider dropping entirely if token budget is tight. Retained here because it reflects the *source* capture rate before encoding, which can diverge under CPU pressure.

---

## 4) Output Format

One compressed object per `getstats` call, grouped by report type:

```json
["getstats", scope, payload, ts]
```

Where `payload` is:

```json
{
  "out_v": [...],
  "out_a": {...},
  "in_a":  {...},
  "in_v":  {...},
  "rtt":   [...],
  "cp":    [...],
  "cq":    {...},
  "ms":    {...}
}
```

### Rules
- **Array types** (`out_v`, `rtt`, `cp`): multiple entries possible per sample. Each array element is one compressed entry.
- **Object types** (`out_a`, `in_a`, `in_v`, `cq`, `ms`): single entry per sample.
- **Omit empty keys**: if a report type has no entries in a given sample, omit the key entirely.
- **Scope determines content**:
  - `0-pub` contains: `out_v`, `out_a`, `rtt`, `cp` (active), `ms`
  - `0-sub` contains: `in_a`, `in_v`, `cp` (active + relay)
  - SFU scope contains: `cq` only

---

## 5) Drop Rules & Edge Cases

### Unconditional drops
- **Timestamp-only entries**: any entry whose only field is `timestamp` → DROP
- **`timestamp` field** within each stat entry → DROP (redundant with envelope `ts`)
- **`remoteId`**, **`localId`**, **`codecId`**, **`ssrc`**, **`mediaType`**, **`kind`**, **`type`** → DROP (internal WebRTC correlation / classification IDs; type is implicit from fingerprint)
- **Media source audio** entries → DROP (always timestamp-only)
- **Top-level `timestamp`** key in the stats object (non-dict value) → DROP

### Deduplication
- **`framesSent`** always equals `framesEncoded` in observed data → DROP `framesSent`, keep `framesEncoded` as `fe`
- **`framesReceived`** vs `framesDecoded`**: keep both — they can diverge under packet loss
- **`discardedPackets`** and **`packetsDiscarded`** are aliases → use `pd` (from `packetsDiscarded`), drop `discardedPackets`

### Sparse fields
- Fields that appear only sometimes (e.g., `pliCount`, `freezeCount`, `nackCount`, `keyFramesDecoded`, `silentConcealedSamples`): **include when present and non-zero delta, omit otherwise**.

### Zero-delta suppression
- After computing deltas, if a counter's delta is `0`, omit that field from the output.
- For gauges, if value is `0`, `null`, or absent, omit the field.

---

## 6) Full Example

### Before: Raw `0-pub` getstats (sample at `ts=1770106218547`)

```json
["getstats", "0-pub", {
  "12620210": {
    "timestamp": 0, "bytesSent": 2343358, "packetsSent": 2199,
    "framesEncoded": 383, "framesSent": 383, "headerBytesSent": 45177,
    "qpSum": 9230, "totalEncodeTime": 2.105, "totalEncodedBytesTarget": 19074426
  },
  "15678729": {
    "timestamp": 1770106217958, "jitter": 0.0009,
    "packetsReceived": 2080, "roundTripTime": 0.013,
    "roundTripTimeMeasurements": 34, "totalRoundTripTime": 0.429
  },
  "27660164": {
    "timestamp": 0, "bytesSent": 1691537, "packetsSent": 2116,
    "framesEncoded": 853, "framesPerSecond": 30, "framesSent": 853,
    "headerBytesSent": 80560, "qpSum": 30032,
    "totalEncodeTime": 3.289, "totalEncodedBytesTarget": 19074426
  },
  "94e74fcb": {
    "timestamp": 0, "bytesSent": 2456292, "packetsSent": 2494,
    "framesEncoded": 781, "framesPerSecond": 30, "framesSent": 781,
    "headerBytesSent": 54576, "qpSum": 21933,
    "totalEncodeTime": 3.192, "totalEncodedBytesTarget": 19074426
  },
  "d9053903": {
    "timestamp": 0, "bytesSent": 128276,
    "packetsSent": 891, "headerBytesSent": 17820
  },
  "b3431ad2": {
    "timestamp": 1770106217687, "jitter": 0.002,
    "packetsReceived": 984, "roundTripTime": 0.012,
    "roundTripTimeMeasurements": 37, "totalRoundTripTime": 0.467
  },
  "21c5c1ea": {
    "timestamp": 1770106217816, "jitter": 0.0006,
    "packetsReceived": 2061, "roundTripTime": 0.012,
    "roundTripTimeMeasurements": 12, "totalRoundTripTime": 0.153
  },
  "mediasource_audio_1{...}": {"timestamp": 0},
  "mediasource_video_0{...}": {
    "timestamp": 0, "frames": 853, "framesPerSecond": 30
  },
  "{5ab326fa-...}": {"timestamp": 0},
  "64f139d4": {"timestamp": 0},
  "a10baabe": {"timestamp": 0},
  "7fd8cbe": {
    "timestamp": 0, "bytesReceived": 28870, "bytesSent": 6962282,
    "currentRoundTripTime": 0.012, "lastPacketReceivedTimestamp": 529681530521,
    "lastPacketSentTimestamp": 529681530537,
    "responsesReceived": 7, "totalRoundTripTime": 0.086
  },
  "8ac7319d": {"timestamp": 0},
  "...": "... 11 more timestamp-only entries ..."
}, 1770106218547]
```

**Raw entry count:** ~25 entries (including ~17 timestamp-only)

### After: Compressed (with deltas from previous sample at `ts=1770106208548`)

```json
["getstats", "0-pub", {
  "out_v": [
    {"bs":1901627,"hbs":35460,"ps":1773,"fe":298,"qp":5443,"tet":1.613,"tebt":9826634},
    {"bs":187814,"hbs":6020,"ps":301,"fe":298,"fps":30,"qp":7623,"tet":1.633,"tebt":9826634},
    {"bs":622365,"hbs":13540,"ps":677,"fe":298,"fps":30,"qp":5120,"tet":1.629,"tebt":9826634}
  ],
  "out_a": {"bs":24504,"hbs":3440,"ps":172},
  "rtt": [
    {"rtt":0.013,"j":0.0009,"pr":680,"trtt":0.130,"rttm":10},
    {"rtt":0.012,"j":0.002,"pr":304,"trtt":0.131,"rttm":10},
    {"rtt":0.012,"j":0.0006,"pr":1793,"trtt":0.130,"rttm":10}
  ],
  "cp": [{"bs":2849206,"br":10444,"rtt":0.012,"rr":2,"trtt":0.025}],
  "ms": {"f":298,"fps":30}
}, 1770106218547]
```

**Compressed entry count:** 5 keys, ~10 sub-entries. Timestamp-only entries eliminated.

### Before: Raw `0-sub` getstats (same timestamp)

```json
["getstats", "0-sub", {
  "3d1ee306": {
    "timestamp": 0, "jitter": 0.005, "packetsReceived": 917,
    "audioLevel": 0.076, "bytesReceived": 123431,
    "concealedSamples": 2858, "concealmentEvents": 3,
    "headerBytesReceived": 11004, "jitterBufferDelay": 50678.4,
    "jitterBufferEmittedCount": 875520,
    "jitterBufferMinimumDelay": 45158.4,
    "jitterBufferTargetDelay": 45158.4,
    "lastPacketReceivedTimestamp": 1770106218529,
    "removedSamplesForAcceleration": 1700,
    "totalAudioEnergy": 1.785, "totalSamplesDuration": 39.78,
    "totalSamplesReceived": 1899840
  },
  "98bc27ac": {
    "timestamp": 0, "jitter": 0.015, "packetsReceived": 4675,
    "bytesReceived": 5120485, "framesAssembledFromMultiplePackets": 587,
    "framesDecoded": 587, "framesPerSecond": 14, "framesReceived": 587,
    "headerBytesReceived": 112200, "jitterBufferDelay": 46.785,
    "jitterBufferEmittedCount": 588,
    "jitterBufferMinimumDelay": 34.756,
    "jitterBufferTargetDelay": 34.756,
    "lastPacketReceivedTimestamp": 1770106218509,
    "qpSum": 85881, "totalAssemblyTime": 6.865,
    "totalDecodeTime": 1.086, "totalInterFrameDelay": 39.202,
    "totalProcessingDelay": 47.792,
    "totalSquaredInterFrameDelay": 2.665
  },
  "8e75ab76": {
    "timestamp": 1770106217807, "bytesSent": 128517,
    "packetsSent": 880, "remoteTimestamp": 1770106214224.49
  },
  "2c507d8e": {
    "timestamp": 1770106217808, "bytesSent": 5130919,
    "packetsSent": 4585, "remoteTimestamp": 1770106214226
  },
  "89f61d59": {
    "timestamp": 0, "bytesReceived": 5462478, "bytesSent": 36364,
    "lastPacketReceivedTimestamp": 529681530529,
    "lastPacketSentTimestamp": 529681530512,
    "responsesReceived": 8, "totalRoundTripTime": 0.094
  },
  "...": "... 19 more timestamp-only entries ..."
}, 1770106218547]
```

### After: Compressed `0-sub`

```json
["getstats", "0-sub", {
  "in_a": {"br":43873,"hbr":3588,"pr":299,"j":0.005,"al":0.076,"tae":0.445,"tsd":10.0,"tsr":480960,"cs":2858,"ce":3,"rsa":1700,"jbd":20928,"jbe":282240,"jbm":14976,"jbt":14976},
  "in_v": {"br":1321902,"hbr":28824,"pr":1201,"j":0.015,"fd":149,"fr":149,"fps":14,"fam":149,"qp":21422,"tdt":0.266,"tifd":9.97,"tsid":0.676,"tat":1.898,"tpd":14.351,"jbd":14.179,"jbe":150,"jbm":11.07,"jbt":11.07},
  "cp": [
    {"br":1423587,"bs":8196,"rr":2,"trtt":0.025},
    {"bs":42279,"ps":276,"rts":1770106214224.49},
    {"bs":1351764,"ps":1203,"rts":1770106214226}
  ]
}, 1770106218547]
```

### SFU scope sample

```json
["getstats", "1-sfu-dpk-frankfurt-vp1-54d1dc529306.stream-io-video.com", {
  "cq": {"s":94.18,"as":98.6,"mos":4.43}
}, 1770106219803]
```
