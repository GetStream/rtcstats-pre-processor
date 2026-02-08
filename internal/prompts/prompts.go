// Package prompts provides importable prompt strings that translate compressed
// field names back to human-readable descriptions. Inject these into LLM
// prompts so the model can interpret abbreviated rtcstats output.
package prompts

// StatsFields maps abbreviated getstats report type and field keys to their meanings.
const StatsFields = `Δ=delta(change since last sample, omitted when 0) G=gauge(snapshot, omitted when 0/null) S=sparse(only present when non-zero)
Report types: out_v=outbound video out_a=outbound audio in_a=inbound audio in_v=inbound video rtt=remote-inbound RTT cp=active candidate pair cp_r=relay candidate pair cq=connection quality(SFU) ms=media source video
Fields: bs=bytesSent(Δ) hbs=headerBytesSent(Δ) ps=packetsSent(Δ) br=bytesReceived(Δ) hbr=headerBytesReceived(Δ) pr=packetsReceived(Δ) fe=framesEncoded(Δ) fd=framesDecoded(Δ) fr=framesReceived(Δ) fps=framesPerSecond(G) f=frames(Δ) fam=framesAssembledFromMultiplePackets(Δ) qp=qpSum(Δ) j=jitter(G,sec) al=audioLevel(G,0-1) tae=totalAudioEnergy(Δ) tsd=totalSamplesDuration(Δ,sec) tsr=totalSamplesReceived(Δ) cs=concealedSamples(ΔS) ce=concealmentEvents(ΔS) rsa=removedSamplesForAcceleration(ΔS) scs=silentConcealedSamples(ΔS) tet=totalEncodeTime(Δ,sec) tebt=totalEncodedBytesTarget(Δ) tdt=totalDecodeTime(Δ,sec) tifd=totalInterFrameDelay(Δ,sec) tsid=totalSquaredInterFrameDelay(Δ) tat=totalAssemblyTime(Δ,sec) tpd=totalProcessingDelay(Δ,sec) jbd=jitterBufferDelay(Δ) jbe=jitterBufferEmittedCount(Δ) jbm=jitterBufferMinimumDelay(Δ) jbt=jitterBufferTargetDelay(Δ) pl=packetsLost(ΔS) pd=packetsDiscarded(ΔS) nk=nackCount(ΔS) kfd=keyFramesDecoded(ΔS) pli=pliCount(ΔS) hfs=hugeFramesSent(ΔS) fzc=freezeCount(ΔS) fzd=totalFreezesDuration(ΔS,sec) fdr=framesDropped(ΔS) rtt=roundTripTime(G,sec) trtt=totalRoundTripTime(Δ) rttm=roundTripTimeMeasurements(Δ) rr=responsesReceived(Δ) rts=remoteTimestamp(G) s=score(G,0-100) as=avgScore(G) mos=mosScore(G,1-5)`

// EventFields maps abbreviated connection event payload keys to their meanings.
const EventFields = `Fields: did=deviceId gid=groupId k=kind w=width h=height en=enabled mu=muted rs=readyState sid=sessionId uid=userId tt=trackType(1=audio,2=video) dir=direction pt=peerType(0=pub,1=sub) mid=mediaLineId mli=sdpMLineIndex ok=success(1/0) dur=durationMs errc=errorCode err=errorMsg rid=correlationId t=type n=count eoc=endOfCandidates fr=fastReconnect(1/0) cap=capabilities bp=bundlePolicy st=permissionState(g/p/d)
Kinds: a=audio v=video | Devices: ai=audioinput vi=videoinput ao=audiooutput
States(int): signaling(stable=0,have-local-offer=1,have-remote-offer=2,closed=3+) iceConn(new=0,checking=1,connected=2,completed=3,disconnected=4,failed=5,closed=6) iceGather(new=0,gathering=1,complete=2) conn(new=0,connecting=1,connected=2,disconnected=3,failed=4,closed=5)`

// SDPDigestFields maps abbreviated SDP digest (sdp_sum) keys to their meanings.
const SDPDigestFields = `sdp_sum fields: type=offer|answer sdp_hash=sha256 bundle_mids=bundledMediaLineIds mline_count=mediaLineCount mid=mediaLineId kind=audio|video|application dir=sendrecv|sendonly|recvonly|inactive rejected=portIsZero codecs=orderedCodecNames sim_rids=simulcastRIDCount tcc=transportWideCCEnabled`

// ScopeReference explains scope string conventions.
const ScopeReference = `Scopes: 0-pub=publisher 0-sub=subscriber sfu:<region>=SFU`

// SamplingReference explains adaptive sampling markers in the output.
const SamplingReference = `Sampling: When adaptive sampling is enabled, getstats events are thinned to every Nth sample. Full resolution is preserved around interesting moments (packet loss, freeze, FPS/jitter/RTT changes). Category value "="=unchanged since last emitted sample (steady-state suppression). Counter deltas in sampled output are accumulated over skipped samples so totals remain correct.`

// FullReference combines all field references into one prompt.
const FullReference = StatsFields + "\n" + EventFields + "\n" + SDPDigestFields + "\n" + ScopeReference + "\n" + SamplingReference
