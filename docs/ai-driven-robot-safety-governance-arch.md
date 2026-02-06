# AI-Driven Robot Safety & Governance Architecture

## 1. Overview

This document describes the **safety strategy, governance model, and execution architecture** for an AI-driven robotic system that supports:

* Cloud-based AI agents (when connectivity exists)
* Offline/onboard decision-making (Jetson-class compute)
* A unified **executor loop** that schedules robot actions via an MCP (Model Context Protocol) server
* Hard real-time control loops that are strictly separated from AI-driven decision loops

The core design principle is **layered safety with explicit authority boundaries**: no AI system—cloud or local—ever directly controls actuators. All actions are mediated, gated, logged, and revocable.

---

## 2. Core Safety Philosophy

### 2.1 Separation of Concerns

| Layer              | Responsibility                       | Timing        | Safety Role               |
| ------------------ | ------------------------------------ | ------------- | ------------------------- |
| Control Loop (CL)  | Actuator control, PID, stabilization | Fixed, fast   | Immediate physical safety |
| Executor Loop (EL) | Decisions, task sequencing           | Variable      | Mission-level safety      |
| Safety Monitors    | Constraints, veto, shutdown          | Continuous    | Absolute authority        |
| AI Agents          | Planning & reasoning                 | Opportunistic | Advisory only             |

Key rule:

> **AI suggests. Safety enforces. Control executes.**

---

## 3. Safety Domains

### 3.1 Hardware Safety (Always Enforced)

* Physical emergency stop (hard power cut)
* Electrical shutdown paths
* Actuator limits (speed, torque, travel)
* Torque-controlled actuation support

These mechanisms do **not** depend on software correctness.

---

### 3.2 Software Safety

* Process watchdogs (hang detection, forced shutdown)
* Heartbeat monitors between subsystems
* Automatic fail-safe outputs on missed heartbeats

---

### 3.3 Runtime Safety Monitors (Always Present)

Runtime monitors operate independently of AI agents:

* Speed checks
* Geofencing
* Obstacle proximity checks
* Human presence detection
* Power and thermal limits

Violations result in:

* Command rejection
* Mode degradation
* Emergency stop (if required)

---

## 4. Control Loop vs Executor Loop

### 4.1 Control Loop (CL)

* Fixed-frequency
* Handles:

  * Motor control
  * PID loops
  * Emergency actuator shutdown
* Rejects any command violating safety config

### 4.2 Executor Loop (EL)

* Variable frequency
* Handles:

  * "Go here"
  * "Do this task"
  * "Stop all"
* Cannot issue low-level motor commands

Each loop enforces **different classes of safety**.

```mermaid
flowchart TD
    EL[Executor Loop]
    CL[Control Loop]
    SM[Safety Monitors]

    EL -->|High-level commands| SM
    SM -->|Validated actions| CL
    CL -->|Actuator signals| Motors
    SM -->|Veto / Shutdown| CL
```

---

## 5. MCP-Based Action Governance

### 5.1 MCP Server Role

The robot hosts an MCP server that:

* Exposes **actions as tools**
* Exposes **state and resources as data**
* Enforces:

  * Parameter bounds
  * Rate limits
  * Authority leases
  * Command TTLs

No agent (cloud or local) bypasses MCP.

---

### 5.2 Conditional Command Execution

All commands are:

1. Checked against safety configuration
2. Evaluated by runtime monitors
3. Logged
4. Executed only if tagged **safe**

```mermaid
sequenceDiagram
    participant Agent
    participant MCP
    participant Safety
    participant Control

    Agent->>MCP: Tool Call
    MCP->>Safety: Validate
    Safety-->>MCP: Approve / Reject
    MCP->>Control: Execute
    Control-->>MCP: Status
    MCP-->>Agent: Result
```

---

## 6. AI / ML Governance

### 6.1 Known AI Failure Modes

* Non-determinism
* False confidence
* Sensitivity to lighting/clutter
* Hallucinated certainty

### 6.2 Mitigations

* Confidence gates on perception outputs
* Safety envelope above AI decisions
* Mandatory decision logging
* Deterministic fallback behaviors

```mermaid
flowchart TD
    AI[AI Inference]
    CG[Confidence Gates]
    SE[Safety Envelope]
    EL[Executor Loop]

    AI --> CG
    CG -->|Low confidence| SafeMode[Degraded Mode]
    CG --> SE
    SE --> EL
```

---

## 7. Executor Loop Modes

### 7.1 Online (Cloud Agent)

* Large LLM
* Full mission context
* Produces tool-call schedules
* Executor Scheduler runs locally

### 7.2 Offline (Onboard / Jetson)

* Deterministic planners (FSM / behavior trees)
* Smaller LLMs (optional)
* Reduced context window
* Same scheduler, same MCP tools

### 7.3 Hybrid

* Cloud handles long-horizon planning
* Jetson handles safety analysis & short-horizon decisions

```mermaid
sequenceDiagram
    participant Human
    participant Cloud
    participant Executor
    participant MCP
    participant Safety

    Human->>Cloud: Mission
    Cloud->>Executor: Plan
    Executor->>MCP: Schedule Tool Calls
    MCP->>Safety: Validate
    Safety->>MCP: OK
    MCP->>Executor: Result
```

---

## 8. Human-in-the-Loop (HITL)

Supported at multiple levels:

* Startup authorization
* High-risk action approval
* On-the-fly mode changes

Humans never directly control actuators—only **authority and intent**.

**Note**: Gorai supports a 'human-drive-only' mode that can be switched into on the fly, which completely bypasses all autonomous controls and allows for human control over actuators directly. 

---

## 9. Auditability & Flight Recorder

### 9.1 Logging Requirements

* Append-only logs
* All decisions traceable
* Hazard → Mitigation → Verification → Deploy chain

### 9.2 Flight Recorder

* Rolling buffer of last N minutes
* Sensor data
* Decisions
* Tool calls
* Safety events

```mermaid
flowchart LR
    Sensors --> Recorder
    Decisions --> Recorder
    MCP --> Recorder
    Safety --> Recorder
```

---

## 10. Cybersecurity

* Secure comms
* Token-based authentication
* Verified senders
* Authority leasing

No command is trusted without identity.

---

## 11. Summary

This architecture ensures:

* AI is powerful but never authoritative
* Safety is layered, explicit, and enforceable
* Cloud and offline modes share the same execution contract
* Every action is auditable, bounded, and revocable

**The executor loop is the invariant.**
The AI is a replaceable advisor.
