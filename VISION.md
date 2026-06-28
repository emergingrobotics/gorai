# Gorai Vision: Capabilities Over NATS

**The robotics platform for the AI era.** Pronounced "go-ray" (like "sting-ray").

---

## The One-Sentence Thesis

> **A robot is a set of capabilities on a message mesh — not a chassis.**
> Sensors are *resources*, actuators are *tools*, and any agent on the mesh can read the world and change it. The capabilities can live on one machine or be scattered across a dozen physical platforms; from the agent's point of view it is one robot.

Gorai gives AI agents the same powers the Model Context Protocol (MCP) gives them — typed **resources** to read, typed **tools** to call, **events** pushed without polling, and **live capability discovery** — but it provides those powers **natively over NATS**, with no MCP server, no JSON-RPC bridge, and no stdio/HTTP plumbing. We call this **NCP — the NATS Capability Protocol** (the *N* is for NATS, the substrate the whole thing rides on).

This document is the north star. Everything in the ecosystem — the core framework, the gateways, the device services — exists to make NCP and the Composite Robot real.

---

## Why We Are Doing This

The Model Context Protocol got one big thing right: it gave language models a clean, typed contract for reaching outside their own context — *resources* to read state, *tools* to take action, *notifications* to be told when something changed. That contract is exactly what an autonomous robot needs. A robot's whole job is to read the world (sensors) and change it (actuators), under the direction of something that reasons (an agent).

But MCP's transports — stdio (a local subprocess) and HTTP+SSE (one client, one server) — were designed for a chatbot talking to a tool server on the same laptop. They do not describe a greenhouse with forty relays, a harbor with three boats, or a field robot whose camera is on a drone and whose gripper is on a rover. They cannot fan out. They cannot reach an edge device behind a flaky link. They cannot let two agents share one set of actuators. They have no audit trail.

NATS can do all of those things, and it was built to. So instead of tunnelling MCP through NATS and keeping a translation layer alive, we take MCP's *ideas* — its capability model — and rebuild them as **first-class NATS citizens.** The capability model survives. The protocol overhead disappears.

> **We keep MCP's capabilities. We drop MCP's interfaces.**

---

## NCP: The Capability Model, Mapped to NATS

MCP defines a small, sharp set of primitives. Each one has a natural, lower-overhead expression as a NATS subject convention. This is the whole of NCP.

| MCP primitive | What it gives an agent | NCP equivalent (pure NATS) |
|---|---|---|
| **Resource** (read-only context) | Pull current state | **Sensor / state.** Snapshot via request/reply on `…<name>.state`; live stream via pub/sub on `…<name>.data`. |
| **Tool** (callable action, typed args) | Do something in the world | **Actuator / command.** Request/reply on `…<name>.command`, arguments validated against a registered JSON Schema. |
| **Notification** (server → client push) | Be told when something happens | **Event.** Fire-and-forget pub/sub on `…<name>.event` — faults, limit switches, threshold crossings. |
| **Tool/resource listing** (`tools/list`) | Know what is available | **Mesh registry.** A NATS JetStream KV catalog of every live capability, queryable by name, type, robot, or capability tag. |
| **`list_changed`** (capabilities changed) | Stay current without polling | **KV watch + heartbeats.** The catalog is watchable; joins and departures arrive as events. The agent's action space updates in seconds. |
| **`initialize` / lifecycle** | Capability handshake | **Announce / heartbeat / TTL.** A node announces on join, heartbeats on an interval, and is reaped on silence; graceful exit publishes a tombstone. |
| **Transport** (stdio, HTTP+SSE) | Move bytes | **NATS.** Request/reply, pub/sub, queue groups, JetStream persistence, leaf nodes to the edge. |

Two primitives carry the weight, and they map onto the two halves of every robot:

- **Resources are sensors.** Read-only. An IMU, a temperature probe, a GPS fix, a camera frame, a battery state of charge. The agent *observes* through resources.
- **Tools are actuators.** Side-effecting. A relay, a motor, a servo, a valve, a thruster, a gripper. The agent *acts* through tools.

Read the world. Reason. Act on the world. Verify. Repeat. That loop is the entire job, and NCP is the contract that makes every step a typed NATS message.

### What this looks like on the wire

NCP rides the Gorai subject convention already used across the platform:

```
gorai.<robot_id>.<component>.<instance>.<suffix>
```

```
gorai.greenhouse.relay.1.command      # tool call     → close relay 1
gorai.greenhouse.sensor.temp.state    # resource read → current temperature
gorai.greenhouse.sensor.temp.data     # resource stream → temperature at 1 Hz
gorai.harbor.thruster.port.command    # tool call     → set port thruster
gorai.harbor.motor.2.event            # event         → "motor 2 fault: overcurrent"
gorai.mesh.announce                   # capability join/leave
gorai.mesh.heartbeat.<service-id>     # liveness
```

Discovery is the existing Gorai mesh: three JetStream KV buckets — `gorai-services` (who is alive, TTL-bounded), `gorai-channels` (what subjects carry what, and their QoS), and `gorai-schemas` (the JSON Schemas that type every tool's arguments and every resource's payload). An agent that can read those three buckets knows the robot's entire capability surface — its complete set of MCP-equivalent resources and tools — without anyone configuring it.

### Why drop the MCP server entirely

The earlier `mcp-over-nats.md` sketch kept a single MCP server as a bridge: the only component that "spoke MCP," translating every call onto NATS. NCP deletes that component. The agent speaks NATS. The capability speaks NATS. There is nothing in between to translate, to restart, to bottleneck, or to lie. The benefits compound:

- **No 1:1 client/server.** NATS fans out by design — one stream to many agents, one queue group across many redundant actuators.
- **Many agents, one mesh.** A perception agent, a safety monitor, and a fleet coordinator can all subscribe and act at once. No agent owns the connection.
- **Physical reach.** A robot on a boat connects to the hub through a NATS leaf node; local control stays local even when the link drops.
- **Built-in audit.** A JetStream stream on `gorai.>` is a complete, replayable record of every command and every reading — who acted, when, and why.
- **One substrate.** Discovery, invocation, streaming, persistence, and audit are all the same technology. There is no second system to learn or operate.

---

## The Big Idea: The Composite Robot

This is the part that is genuinely new, and it falls straight out of treating a robot as capabilities on a mesh rather than as a box of wires.

> **A robot need not be a single physical unit. It can be many physical platforms acting as one.**

In conventional robotics, "the robot" is the chassis — one computer, one bus, one power rail, one bounded thing. Capabilities are trapped inside it. In Gorai, **a robot is a logical scope** (`robot_id`) over a set of capabilities registered in the mesh. Where those capabilities physically live is an implementation detail. The agent addresses a capability *by name*; NATS routes the message to wherever it actually runs.

Once location stops mattering, the chassis stops mattering. A single logical robot can be assembled from:

- a **ground rover** that owns mobility and a manipulator arm,
- an **aerial scout** that owns an overhead camera resource and a GPS fix,
- a **fixed sensor mast** that owns wind, temperature, and a wide-angle camera,
- a **compute box** in a weatherproof case that owns vision and planning services.

Four physical platforms. One mesh. One `robot_id`. To the agent running the patrol loop, it is **one robot** with mobility, an arm, two cameras, GPS, weather, and onboard vision — even though no single device has all of those. The drone's camera is just another resource; the rover's arm is just another tool. They are siblings in the same capability catalog.

**Composition is a choice, not magic.** Every robot starts as a *base robot* — the explicit set of capabilities its definition declares and the CLI builds. Spanning outward to adopt capabilities hosted on other platforms is opt-in: a single discovery switch turns runtime composition on. Off, you get exactly the robot you defined; on, the robot grows to include whatever it is authorized to discover on the mesh. And authorization is literal — a platform must hold valid credentials for the robot's NATS account before it can be seen or adopted, so the credential set *is* the robot's boundary.

### Why this matters

- **Composition replaces integration.** You do not build one robot that does everything. You bring capable platforms into a shared scope and the robot *emerges* from their union. Adding a capability means powering on a platform that announces it.
- **Runtime re-composition.** Platforms join and leave while the robot is running. The drone lands to swap batteries and its camera resource disappears; the agent's action space shrinks; the patrol degrades gracefully; the drone returns and the capability comes back — all through the same `list_changed` / KV-watch machinery, no restart.
- **Fluid boundaries.** The same aerial scout can be a capability of the rover-robot during a survey, then leave and become its own robot, then join a *fleet* robot that spans the whole site. A robot is a membership, not a manufacture.
- **Heterogeneous by default.** A Raspberry Pi rover, an RP2040-driven actuator board behind a serial gateway, and an x86 compute box are peers on the mesh. Each contributes the capabilities it is good at. Nothing has to be rewritten to live in one chassis.

### Distributed-systems thinking, applied to robots

This is the deliberate move at the heart of Gorai: **treat a robot as a distributed system, because that is what it already is.** The discipline that built reliable cloud systems — service discovery, location transparency, health checking, graceful degradation, fan-out messaging, event sourcing, replay — is exactly the discipline a multi-platform robot needs. We are not inventing robotics middleware from scratch; we are pointing proven distributed-systems patterns at actuators and sensors.

| Distributed-systems concept | In the Composite Robot |
|---|---|
| Service registry | The mesh KV catalog of resources and tools |
| Service discovery | An agent finding a sensor or actuator by name/tag |
| Location transparency | Calling a tool without knowing which platform hosts it |
| Health checks / heartbeats | Capability liveness and the TTL reaper |
| Load balancing | NATS queue groups across redundant actuators |
| Pub/sub fan-out | One sensor stream feeding many agents |
| Event sourcing / replay | JetStream audit of every command and reading |
| Graceful degradation | A platform leaving shrinks the action space, not the robot |

---

## NATS as the Perfect Fan-Out Fabric

A physical world that several minds want to observe and act on needs *fan-out*, and fan-out is what NATS does best.

- **One sensor, many readers.** A camera publishes once to `gorai.<robot>.camera.front.data`. A vision agent, a recorder, a safety monitor, and a dashboard all receive it. The publisher neither knows nor cares how many are listening.
- **Many actuators, one command class.** Redundant relay nodes join the queue group `actuator-workers`; NATS load-balances incoming tool calls across whichever node is free. Scale by powering on another node.
- **Many agents, concurrent.** Several agents share the same capability surface at once. Tool calls are request/reply, so each gets its own typed acknowledgement; resource streams are pub/sub, so each gets every reading.
- **Edge and core, one logical bus.** Leaf nodes extend the mesh to a robot in the field. Local traffic stays local and low-latency; only what the hub subscribes to crosses the link; the robot keeps running if the link does not.

The fan-out is not a feature we built. It is the messaging fabric's native behavior, inherited for free the moment capabilities became NATS subjects instead of MCP endpoints.

---

## Safety Lives at the Capability, Never at the Agent

An agent reasons about goals. It can hallucinate an argument, misread context, or be steered by a hostile prompt. So **the agent is never trusted to be safe.** Every physical constraint is enforced at the capability node — the thing actually touching hardware — regardless of what arrived on the wire.

- **Clamp every value** before it reaches a pin or a PWM channel.
- **Enforce interlocks** — e-stop, thermal limits, mechanical limits — in the node handler, not the planner.
- **Rate-limit** to what the hardware can physically survive.
- **Return structured errors.** "speed out of range" is something an agent can reason about and recover from; a silent failure is not.
- **Audit everything** via a JetStream stream on `gorai.>`, so every command and its outcome is recoverable after the fact.

> **Autonomy without replay is folklore.** Action logs, state streams, and replay are first-class platform concerns, not optional add-ons. This is non-negotiable precisely because the thing issuing commands is allowed to be wrong.

---

## What This Means for the Architecture

The full stack, top to bottom:

```
Agents (one or many — perception, safety, coordination)
  │  read resources, call tools, receive events — all as NATS messages (NCP)
  ▼
NATS mesh  ──  discovery (KV catalog) · fan-out (pub/sub, queue groups) · audit (JetStream) · reach (leaf nodes)
  │
  ├─ Capability node on Platform A (rover):   mobility tool, arm tool
  ├─ Capability node on Platform B (drone):   overhead-camera resource, GPS resource
  ├─ Capability node on Platform C (mast):    weather resources, wide camera resource
  └─ Capability node on Platform D (compute): vision service, planning service
        │
        └─ enforces safety, drives hardware
```

Each layer has one responsibility. **Agents reason. NATS distributes. Capability nodes enforce safety and drive hardware.** New capabilities self-announce; departed ones self-expire; the agents' view of what the robot can do stays current without polling, restart, or manual configuration — across as many physical platforms as the robot happens to span today.

### How it composes with what already exists

NCP and the Composite Robot are not a rewrite. They are the *name and the purpose* of machinery Gorai already has:

- The **subject convention** (`gorai.<robot>.<component>.<instance>.<suffix>`) is the NCP wire format.
- The **mesh service discovery** (the `gorai-services` / `gorai-channels` / `gorai-schemas` KV buckets) is NCP's capability catalog — the `tools/list` equivalent.
- **Dynamic discovery and proxies** are how a capability on one platform appears as a first-class resource or tool on another — the Composite Robot at runtime.
- **Protocol gateways** (serial/GSP, Modbus, MQTT/Tasmota) bring non-NATS devices onto the mesh as native capabilities, so a $5 microcontroller or an off-the-shelf smart plug becomes a tool an agent can call.
- **Leaf nodes and clustering** are how the mesh — and therefore a single robot — spans machines, sites, and links.

The pieces were already pointed at this. This document states the target they were aiming for.

---

## Scope

**In scope:** AI and robotics meeting in the physical world — agents using NCP resources (sensors) and tools (actuators) over NATS to perceive and act; the Composite Robot spanning multiple physical platforms; the core framework, gateways, and device services that make those real.

**Out of scope:** software-only coding agents and standalone LLM-serving infrastructure. Gorai is about the physical world. Where a robot needs reasoning, the reasoning agent is a *client* of the mesh — it reads resources and calls tools like any other agent — but building general-purpose software-development agents is not what this project is.

---

## The Bet

Robotics has spent a decade asking how to wire one very capable machine. The AI era is asking a different question: *how do we let a mind safely observe and act on the physical world, at whatever scale the task demands?* The answer is not a bigger chassis. It is a capability mesh — MCP's contract, NATS's fabric, and a robot defined by what it can do rather than where its parts are bolted.

**Read the world as resources. Change it through tools. Let the robot be as large as it needs to be.**
