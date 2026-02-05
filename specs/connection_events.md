# WebRTC / SFU Event Log Token-Optimized Payload Spec (LLM-friendly)

**Goal:** Minimize tokens without losing diagnostic accuracy/correlation.  
**Constraint:** **Do not change event names** (eventName is preserved verbatim).  
**Note:** SDP is **not** specified here. Wherever SDP appears, replace it with `sdp_sum` (per your existing SDP summary spec).

---

## 0) Common Log Envelope (applies to all events)

Each record remains an array:

`[eventName, scope, payload, ts]`

### Envelope rules
- `eventName` **unchanged**
- `scope` (2nd element): keep, but compress long hostnames via hashing or short aliasing.
  - Example: `"0-sfu-dpk-frankfurt-vp1-...stream-io-video.com"` → `"sfu:frankfurt-vp1"` or `"h:<hash>"`
- `ts` (4th element): prefer `dt` (delta ms from first event) if you can; otherwise keep epoch ms.

### Optional (high value, low cost)
If you can add fields inside `payload`, use:
- `rid`: correlation id for request→success/failure pairs (offer/answer, setLocal/Remote, etc.)
- `ok`: 1/0 for success/failure events (or omit when implied by eventName)
- `dur`: duration in ms (when measurable)

---

## 1) Field Renaming Map (payload keys only)

Use these short keys to reduce tokens:

| Original | Short |
|---|---|
| deviceId | did |
| groupId | gid |
| kind | k |
| label | (drop) |
| width | w |
| height | h |
| enabled | en |
| muted | mu |
| readyState | rs |
| sessionId | sid |
| unifiedSessionId | (drop or usid) |
| userId / user_id | uid |
| trackType / track_type | tt |
| direction | dir |
| timestamp | (drop in payload; ts already in envelope) |
| iceCandidate / candidate | (drop string; use counts/summary) |
| sdp | (replace with `sdp_sum`) |

### Enum conventions (recommended)
- media kind: `a` = audio, `v` = video
- device kind: `ai` audioinput, `vi` videoinput, `ao` audiooutput, `vo` videoinput (rare)
- directions: `in`, `out`
- booleans: `1/0` (not `true/false`)

---

## 2) Event Specs

For each event:
- **Keep:** fields that must remain
- **Strip:** fields to remove
- **Transform:** how to compress/normalize
- **Example payload:** illustrative minimal payload (event name unchanged)

---

### navigator.mediaDevices.enumerateDevices

**Keep**
- device counts by kind: `ai`, `vi`, `ao` (and `vo` if ever appears)
- `hl` (has labels): 1 if any label non-empty, else 0

**Strip**
- full device list
- `deviceId`, `groupId`, `label` for each device

**Transform**
- Count occurrences of each `kind`
- `hl = any(label != "")`

**Example payload**
```json
{"ai":1,"vi":1,"ao":0,"hl":0}
```

---

### navigator.mediaDevices.getUserMedia.<int>

**Keep**
- requested media flags: `a` and/or `v` as 1/0
- constraints only when non-default or relevant
  - audio toggles: `agc`, `ns`, `ec`
  - video dimensions: `w`, `h` (or preset enum)

**Strip**
- deep nesting (`audio:{...}`, `video:{...}`) if you can flatten
- default/empty fields

**Transform**
- flatten audio constraints
- keep only keys that deviate from defaults

**Example payloads**
```json
{"a":1,"agc":1,"ns":1,"ec":1}
```
```json
{"v":1,"w":1280,"h":720}
```

---

### navigator.mediaDevices.getUserMedia.<int>.OnSuccess

**Keep**
- media kinds present + count: `a` and/or `v` as 1/0 (or `t:{a:1,v:1}`)
- optional: `tid` hashed if you need later correlation
- optional: non-default states (`en`, `mu`, `rs`) only if not the common case

**Strip**
- stream id UUID
- full track UUIDs (unless hashed)
- track `label` (token-heavy, PII-ish)

**Transform**
- `tid` = short hash of track id (if retained)
- `rs` enum: `l` live, `e` ended

**Example payload**
```json
{"a":1}
```

---

### navigator.mediaDevices.getUserMedia.<int>.OnFailure

**Keep**
- `ok:0`
- `dur` (if available)
- `errc` (error class / code)
- `err` (short message, capped)

**Strip**
- stacks / long messages

**Example payload**
```json
{"ok":0,"errc":"NotAllowedError","err":"permission denied"}
```

---

### permissions.query(microphone)

**Keep**
- `st` permission state: `g/p/d` (granted/prompt/denied)

**Strip**
- everything else

**Example payload**
```json
{"st":"p"}
```

---

### permissions.query(camera)

**Keep**
- `st` permission state: `g/p/d` (granted/prompt/denied)

**Strip**
- everything else

**Example payload**
```json
{"st":"g"}
```

---

### navigator.mediaDevices.setSinkId

**Keep**
- `ok` (if you log failure)
- `sink` hashed (`h:<hash>`)

**Strip**
- raw sink id

**Example payload**
```json
{"ok":1,"sink":"h:9f3a"}
```

---

### setUseWebAudio

**Keep**
- value as `v` (1/0) or raw `0/1`

**Example payload**
```json
0
```

---

### signal.ws.open

**Keep**
- host alias/hash: `h`
- optional: `ok`
- optional: `dur` (connect latency)

**Strip**
- `{isTrusted:true}`

**Transform**
- `h` derived from hostname → short region/cluster or hash

**Example payload**
```json
{"h":"sfu:frankfurt-vp1","ok":1}
```

---

### joinRequest

**Keep (minimum)**
- `sid` (sessionId)
- `fr` (fastReconnect as 1/0) if not default
- `cap` (capabilities) as bitmask or small list
- client environment, compact:
  - `sdk`: `[type,"1.31.5"]`
  - `os`:  `["win","10","amd64"]`
  - `br`:  `["ff","147"]`

**SDP**
- Replace `publisherSdp` and `subscriberSdp` with:
  - `pub_sdp_sum`
  - `sub_sdp_sum`

**Strip**
- `token` (never send to LLM)
- user name/image/custom payloads
- full requestPayload wrapper fields (`oneofKind`, etc.)
- empty arrays/objects (`preferredPublishOptions:[]`, etc.)

**Example payload**
```json
{
  "sid":"3490..d2ea",
  "sdk":[1,"1.31.5"],
  "os":["win","10","amd64"],
  "br":["ff","147"],
  "cap":1,
  "fr":0,
  "pub_sdp_sum":{...},
  "sub_sdp_sum":{...}
}
```

---

### create  (PeerConnection create)

**Keep**
- `bp` (bundlePolicy) only if not constant
- ICE server summary:
  - `ice`: `{turn:<n>,turns:<n>,tcp:<n>,udp:<n>,hosts:<n>}`

**Strip**
- `iceServers[].username`
- `iceServers[].credential`
- full URL strings & query params

**Transform**
- classify each URL by scheme and transport
- `hosts` count distinct hostnames (hashed optionally)

**Example payload**
```json
{"bp":"mb","ice":{"turn":3,"turns":1,"tcp":2,"udp":2,"hosts":2}}
```

---

### negotiationneeded

**Keep**
- no payload required (event itself is the signal)
- optional: `sig` state enum if cheaply available

**Example payload**
```json
null
```

---

### Create follow up

**Keep**
- `t` follow-up type enum (implementation-specific)
- `rid` if it correlates to an offer/answer flow
- `pc` if you have multiple PCs (or rely on scope)

**Strip**
- long textual descriptions

**Example payload**
```json
{"t":2,"rid":"o1"}
```

---

### createOffer

**Keep**
- optional: `rid` (correlation id)

**Strip**
- `[null]`, `[]` if meaningless

**Example payload**
```json
{"rid":"o1"}
```

---

### createOfferOnSuccess

**Keep**
- `t` type: `"offer"` or `"o"`
- `sdp_sum`
- optional: `cand_in_sdp` 1/0

**Strip**
- full SDP string

**Example payload**
```json
{"t":"o","sdp_sum":{...},"cand_in_sdp":0}
```

---

### createAnswer

**Keep**
- optional: `rid`

**Example payload**
```json
{"rid":"a1"}
```

---

### createAnswerOnSuccess

**Keep**
- `t` type: `"answer"` or `"a"`
- `sdp_sum`
- optional: `cand_in_sdp` 1/0

**Strip**
- full SDP string

**Example payload**
```json
{"t":"a","sdp_sum":{...},"cand_in_sdp":1}
```

---

### setLocalDescription

**Keep**
- `t` type (`offer`/`answer`)
- `sdp_sum`
- optional: `ok`, `dur`

**Strip**
- full SDP

**Example payload**
```json
{"t":"offer","sdp_sum":{...}}
```

---

### setLocalDescriptionOnSuccess

**Keep**
- optional: `dur`

**Example payload**
```json
null
```

---

### signalingstatechange

**Keep**
- state as enum (recommended) or short string

**Transform**
- map to small ints:
  - stable=0, have-local-offer=1, have-remote-offer=2, closed=3, etc.

**Example payload**
```json
0
```

---

### icegatheringstatechange

**Keep**
- state enum: new=0, gathering=1, complete=2

**Example payload**
```json
2
```

---

### iceconnectionstatechange

**Keep**
- state enum:
  - new=0, checking=1, connected=2, completed=3, disconnected=4, failed=5, closed=6

**Example payload**
```json
1
```

---

### connectionStateChange

**Keep**
- state enum:
  - new=0, connecting=1, connected=2, disconnected=3, failed=4, closed=5

**Example payload**
```json
2
```

---

### onicecandidate

**Keep**
- counts only:
  - per-event: `{"n":1}` and aggregate externally
  - end-of-candidates marker: `{"eoc":1}` when detected

**Strip**
- candidate strings
- `usernameFragment`, `sdpMid`, `sdpMLineIndex` (unless you need `mid`, then keep only `mid`)

**Example payloads**
```json
{"n":1}
```
```json
{"eoc":1}
```

---

### setRemoteDescription

**Keep**
- `t` type (`offer`/`answer`)
- `sdp_sum`
- optional flags extracted from SDP summary if you already include them (e.g., `ice_lite`, `eoc`)

**Strip**
- full SDP

**Example payload**
```json
{"t":"answer","sdp_sum":{...}}
```

---

### setRemoteDescriptionOnSuccess

**Example payload**
```json
null
```

---

### addIceCandidate

**Keep**
- `n` count (usually 1)
- failures: `ok:0`, `errc`, short `err`
- optional: `dur`

**Strip**
- candidate string

**Example payload**
```json
{"n":1}
```

---

### addIceCandidateOnSuccess

**Example payload**
```json
null
```

---

### IceTrickle

**Keep**
- `pt` (peerType 0/1)
- `sid`
- candidate summary only:
  - `c`: `{t:"host|srflx|relay", tr:"udp|tcp", mid:"0"}` (mid optional)
- OR just `{"n":1}` and aggregate by `pt`

**Strip**
- `iceCandidate` string (and the nested JSON string)
- raw IP/ports

**Example payload**
```json
{"pt":0,"sid":"3490..d2ea","c":{"t":"relay","tr":"udp","mid":"0"}}
```

---

### UpdateMuteStates

**Keep**
- mute bitset or compact map:
  - `mu`: `{a:0/1, v:0/1}` (include only changed)
- optional: `sid`

**Strip**
- array `muteStates:[{trackType,...}]`

**Example payload**
```json
{"mu":{"v":0}}
```

---

### connectionQualityChanged

**Keep**
- `q` quality enum (implementation-specific; typically 0–5)
- optional: `dir` if you have uplink/downlink

**Strip**
- raw stats dumps

**Example payload**
```json
{"q":4}
```

---

### UpdateSubscriptions

**Keep**
- `sid` (local session)
- compact track requests:
  - per entry: `{u:<uid>, tt:<1|2>, wh:[w,h]}` (wh optional)

**Strip**
- remote `sessionId` if redundant
- nested/verbose fields

**Example payload**
```json
{"sid":"3490..d2ea","tr":[{"u":"VGNGZ9","tt":2,"wh":[1001,563]}]}
```

---

### sfu.track.mapping follow-up

**Keep**
- `dir` (`in`/`out`)
- `tt` (audio/video enum)
- codec short: `c` (e.g., `"opus"`, `"vp8"`)
- optional: `rid` for simulcast layers (q/h/f)
- `s` SSRC hashed or truncated
- `uid` (or one canonical participant id)

**Strip**
- participant session ids if redundant
- verbose track_type names (`TRACK_TYPE_AUDIO`) → enum
- codec strings with trailing punctuation (normalize `"Opus:"` → `"opus"`)

**Example payload**
```json
{"dir":"out","tt":2,"c":"vp8","rid":"q","s":"h:2c91","uid":"5WOKB6"}
```

---

### SetPublisher

**Keep**
- `sid`
- `sdp_sum`
- bounded track summary:
  - `tr`: `[{mid,tt,c,sc}]`

**Strip**
- full SDP
- UUID-heavy `trackId` (hash if needed)
- codec subfields that are default/empty

**Example payload**
```json
{
  "sid":"3490..d2ea",
  "sdp_sum":{...},
  "tr":[{"mid":"0","tt":2,"c":"vp8","sc":[["q",375,320,180],["h",750,640,360],["f",1500,1280,720]]}]
}
```

---

### SetPublisherResponse

**Keep**
- `sid` (if present)
- `sdp_sum`
- optional: `ice_lite` if your sdp_sum exposes it

**Strip**
- full SDP
- empty fields (`sessionId:""`)

**Example payload**
```json
{"sdp_sum":{...}}
```

---

### SendAnswer

**Keep**
- `sid`
- `sdp_sum`

**Strip**
- full SDP

**Example payload**
```json
{"sid":"3490..d2ea","sdp_sum":{...}}
```

---

### ontrack

**Keep**
- `k` kind: `a` or `v`
- `mid` (preferred) if available

**Strip**
- track UUID
- stream id string

**Example payload**
```json
{"k":"a","mid":"0"}
```

---

## 3) Default Omission Rules (token savers)

- Omit keys when value is:
  - `null`, `""`, `[]`, `{}`, `false`, `0` **and** the key is known-default
- Do not repeat invariant fields every event:
  - SDK/OS/Browser: log once per session (e.g., in `joinRequest`) unless changes
- Do not log secrets to LLM:
  - JWTs, TURN usernames/credentials, auth query params

---

## 4) Candidate + SDP Replacement Summary

- Any field containing SDP → replace with:
  - `sdp_sum` (per your existing SDP summary spec)
- Any field containing ICE candidate strings → replace with:
  - `{"n":<count>}` and `{"eoc":1}` markers and/or `{"c":{t,tr,mid}}` classification

---

## 5) Minimal “Gold” Correlation (recommended)

To preserve flow reasoning for the LLM with minimal tokens:
- include `rid` on:
  - createOffer ↔ createOfferOnSuccess ↔ setLocalDescription ↔ setLocalDescriptionOnSuccess
  - setRemoteDescription ↔ setRemoteDescriptionOnSuccess
  - createAnswer ↔ createAnswerOnSuccess ↔ setLocalDescription (sub)
- include `pc` implicitly via `scope` (`0-pub`, `0-sub`) — no extra field needed