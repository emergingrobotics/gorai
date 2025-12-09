# Introduction

> **GoRAI** is pronounced "go-ray" (like "sting-ray")

## About This Book

*GoRAI: Building Modern Robots with Go and NATS* is a practical guide to a new approach in robotics software. Whether you're a software developer curious about robotics, a robotics enthusiast tired of fighting complex frameworks, or an experienced engineer looking for something better, this book will get you building.

We wrote this book because we believe robotics software should be simpler. The tools exist—Go's elegant concurrency, NATS's battle-tested messaging, modern AI accelerators—but nobody had put them together in a way that prioritized developer experience. GoRAI is our answer, and this book is your guide to using it.

## Who We Are

### Greg Herlein

My path to robotics took a circuitous route through some of the most demanding technical environments you can imagine.

I started my career as a US Navy Submarine Nuclear Power Plant Operator and then Supervisor. When you're responsible for a nuclear reactor hundreds of feet underwater, you learn quickly that systems must be simple enough to understand completely, robust enough to never fail, and designed so that the right action is the obvious action. Those lessons never left me.

After the Navy, I spent decades in Silicon Valley leading engineering teams at companies you've heard of—Rackspace, Cisco, AWS—and plenty of startups you haven't. I built distributed systems before "distributed systems" was a buzzword. I learned what works at scale and what doesn't.

But robotics was always my passion on the side. I coached middle school and high school robotics teams in FIRST LEGO League and VEX competitions, watching students struggle with the same software complexity that frustrated professional engineers. At home, I built robots for fun—and ran into those same frustrations myself.

GoRAI grew from a simple question: why is robotics software so much harder than it needs to be? The distributed systems lessons from my career, the simplicity requirements from nuclear power, and the accessibility needs from coaching young roboticists—they all pointed to the same answer. We needed something new.

### Luca Herlein

I grew up building robots. Not as a hobby - I picked that up later — as the thing I did from elementary school through college.

My FIRST LEGO League team made it to the World Championships. I spent years in VEX competitions, learning what it takes to build machines that actually work under pressure. Eight years of competition robotics teaches you things that textbooks can't: that the simple solution usually beats the complex one, that testing matters more than theory, and that the robot that runs reliably beats the robot that runs impressively (sometimes).

I studied Aerospace Engineering at CU Boulder, where I learned the formal foundations—dynamics, control systems, embedded programming. I served as Aerodynamics Lead Engineer on the university's 2021-22 Design Build Fly (DBF) competition team, applying those foundations to aircraft that had to actually fly. But honestly, the competition experience—from FLL through DBF—taught me more about building things that work than any textbook. Academic exercises have known solutions. Competition robots and aircraft face unknown challenges with hard deadlines.

## Why We Wrote This Together

A robotics framework needs two perspectives: the software architect who thinks in distributed systems and long-term maintainability, and the roboticist who thinks in actuators and sensors and "will this work when it matters."

Greg brings decades of building systems that scale and survive. Luca brings years of building robots that compete and win. GoRAI exists at the intersection—software engineering rigor applied to practical robotics.

This book reflects both perspectives. The architectural discussions come from hard-won experience with distributed systems. The practical examples come from actually building robots. When we disagree (and we do), we usually find that both viewpoints have merit—and the synthesis is better than either alone.

## What You'll Learn

By the end of this book, you'll understand:

- **The GoRAI mental model**: How to think about robot software as distributed systems
- **NATS messaging**: Pub/sub, request/reply, and streaming for robotics
- **Component architecture**: Sensors, actuators, cameras—building blocks that compose
- **Service design**: Vision, navigation, and custom capabilities
- **Testing strategies**: From unit tests to hardware validation
- **AI integration**: Running ML models on edge hardware
- **Project organization**: Structuring code that grows with your robot

More importantly, you'll have built things. The hello-sensor example runs real code. The custom component chapter produces working drivers. The testing chapter creates tests that actually catch bugs.

## How to Read This Book

**If you're new to robotics**: Read sequentially. Each chapter builds on the previous. By Chapter 9, you'll understand a complete working system.

**If you're experienced with ROS/ROS 2**: Skim Chapters 1-2 for the philosophical differences, then dive into Chapter 3 (NATS) and Chapter 9 (hello-sensor). The patterns will feel familiar; the simplicity will feel liberating.

**If you're a Go developer exploring robotics**: Chapter 2 (architecture) and Chapter 4-6 (components) will orient you. The code will feel natural; the domain concepts will be new.

**If you just want to build something**: Start with Chapter 8 (environment setup) and Chapter 9 (hello-sensor). Get code running, then circle back to understand why it works.

## A Note on Style

We write the way we talk. Technical concepts deserve clear explanations, not academic obfuscation. Code examples are complete and runnable, not excerpts that require imagination to compile.

When we don't know something, we say so. When multiple approaches work, we explain the tradeoffs rather than pretending one is obviously correct. Robotics is hard enough without authors pretending otherwise.

## Let's Build

The best way to learn robotics is to build robots. The best way to learn GoRAI is to use it.

Fire up your terminal. Clone the repository. Let's get started.

---

*Greg Herlein & Luca Herlein*
*2024*
