# Work in Progress Documentation

This directory contains active development documentation for the Minitest support implementation.

## Document Structure

### 📋 Strategy & Planning

| Document | Purpose | Last Updated |
|----------|---------|--------------|
| [minitest-prd.md](minitest-prd.md) | **Product Requirements Document**<br>High-level goals, success criteria, and phased approach for adding Minitest support to Plur. The "why" behind the implementation. | 2025-06-22 |
| [minitest-todo.md](minitest-todo.md) | **Task Tracker** *(Single source of truth)*<br>All implementation tasks with checkboxes. Organized by phase with links to relevant deep dives. Check here for current progress. | 2025-06-22 |

### 🔬 Technical Deep Dives

| Document | Purpose | Last Updated |
|----------|---------|--------------|
| [test-event-architecture.md](test-event-architecture.md) | **Event Architecture & Design**<br>Comprehensive architecture for the event-based refactoring. Includes current state analysis, TestEvent types, parser interface, accumulator design, and migration strategy. Core technical reference. | 2025-06-22 |
| [minitest-implementation-guide.md](minitest-implementation-guide.md) | **Implementation Guide**<br>Practical implementation details for Minitest support. Documents critical issues (streaming fix), performance characteristics, lessons learned, and debugging tips. Essential for understanding how Minitest was integrated. | 2025-06-22 |

### 📁 Reference Files

| File | Purpose |
|------|---------|
| [plur-json-rows-output.log](plur-json-rows-output.log) | Sample RSpec JSON output for parser development |

## Quick Start

1. **Understand the goal**: Read [minitest-prd.md](minitest-prd.md)
2. **Check progress**: Review [minitest-todo.md](minitest-todo.md)
3. **Dive deep**: Explore technical docs as needed

## Current Status

🚧 **Phase 3: Event-Based Refactoring** - In Progress

Key accomplishments:
- ✅ Framework abstraction complete
- ✅ Basic Minitest support working
- ✅ Streaming output fixed
- 🔄 Event parser integration in progress

Next major milestone: Complete parser integration into runners