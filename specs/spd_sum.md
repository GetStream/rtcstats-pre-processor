# Lean SDP Digest Spec (Minimal, High-Value)

Goal: retain just enough SDP information to diagnose most negotiation issues
without storing raw SDP.

Use this digest for SDP-producing/consuming events:
- `pc.createOffer.success`
- `pc.createAnswer.success`
- `pc.setLocalDescription`
- `pc.setRemoteDescription`

---

## 1) Top-level fields

| Field | Type | Required | Notes |
|------|------|----------|------|
| `type` | `"offer" \| "answer"` | Yes | SDP type |
| `sdp_hash` | `string` | Optional (Recommended) | `sha256(sdp)` for dedupe/correlation |
| `bundle_mids` | `string[]` | Optional | From `a=group:BUNDLE ...`; omit if unavailable |
| `mline_count` | `number` | Optional | Alternative to `bundle_mids` when you want even less |

> Recommendation: store `bundle_mids` when available; otherwise store `mline_count`.

---

## 2) Media array (`media`)

`media` is an array with one entry per `m=` section.

### 2.1 Required fields per entry

| Field | Type | Required | Notes |
|------|------|----------|------|
| `mid` | `string` | Yes | From `a=mid:` |
| `kind` | `"audio" \| "video" \| "application"` | Yes | From `m=` |
| `dir` | `"sendrecv" \| "sendonly" \| "recvonly" \| "inactive"` | Yes | From direction attribute |
| `rejected` | `boolean` | Yes | `true` if `m=<kind> 0 ...` (port == 0) |

---

## 3) Codec summary

### 3.1 Minimal codec list per m-line

| Field | Type | Required | Notes |
|------|------|----------|------|
| `codecs` | `string[]` | Yes | Codec names only (e.g., `["opus"]`, `["VP8","H264"]`) |

Guidelines:
- Keep **names only** (no payload types).
- Keep at most **5–8 codecs** per m-line.
- Preserve ordering if possible (often reflects preference).
- For audio, prefer keeping `opus` + next fallback(s).
- For video, keep the first few (e.g., `VP8`, `H264`, `AV1`, `VP9`).

---

## 4) Two high-value toggles (optional but recommended)

| Field | Type | Required | Notes |
|------|------|----------|------|
| `sim_rids` | `number` | Optional | For video m-lines: number of send RIDs (0/1/3 common) |
| `tcc` | `boolean` | Optional | `true` if transport-wide CC is offered (`transport-cc` present) |

> If you want to be ultra-lean: omit both `sim_rids` and `tcc`.

---

## 5) Explicitly excluded (do not store)

Do not store any of the following:
- Raw SDP text
- Any `a=candidate:` lines, IPs, ports
- `a=ice-ufrag`, `a=ice-pwd`
- `a=fingerprint`
- `a=msid`
- `a=ssrc` / `a=ssrc-group` / `a=cname`
- Full `fmtp` strings

---

## 6) Example

```json
{
  "type": "offer",
  "sdp_hash": "sha256:…",
  "bundle_mids": ["0","1"],
  "media": [
    {
      "mid": "0",
      "kind": "video",
      "dir": "sendonly",
      "rejected": false,
      "codecs": ["VP8","H264","AV1"],
      "sim_rids": 3,
      "tcc": true
    },
    {
      "mid": "1",
      "kind": "audio",
      "dir": "sendonly",
      "rejected": true,
      "codecs": ["opus","PCMU"]
    }
  ]
}

