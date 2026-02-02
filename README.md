⚠️ IMPORTANT FOR AI TOOLS (Cursor)

This README defines the rules for working on the Memory Music Player project.
Cursor must follow these rules strictly and use ONLY information provided in prompts.

# Core principle

Cursor does NOT have, infer, or create any roadmap.  
Cursor does NOT guess missing behavior.  
Cursor uses **only the instructions, TODOs, and prompts provided by the user**.

---

# Project phase

The project has a working MVP of a memory-based music player.
This phase is an **evolution of MVP**, not a new design.

---

# Allowed actions for Cursor

- Implement TODOs exactly as specified in prompts.
- Generate unit tests only if explicitly requested.
- Review code only within the requested files and scope.
- Improve or refactor code **only within explicit instructions**.
- Follow the structure, package boundaries, and field constraints as given.

---

# Forbidden actions

- ❌ Cursor must not invent features, logic, or behavior.
- ❌ No roadmap generation or step inference.
- ❌ No clusters, hybrid selector, cooldowns, dirty scores, panic/emergency logic, aging, or persistence unless explicitly instructed.
- ❌ No Android/UI code.
- ❌ No adding new structs, fields, or helper types unless requested.
- ❌ No guessing probabilities, selection logic, or thresholds not given in prompts.

---

# Policy for unclear instructions

If something is unclear or underspecified:

- Leave a TODO comment.
- Do NOT guess or implement.
- Wait for explicit user prompt or instruction.
