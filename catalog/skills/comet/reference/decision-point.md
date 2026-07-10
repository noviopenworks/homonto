# Decision Point Protocol

Canonical path: `comet/reference/decision-point.md`

This protocol is shared by all comet sub-skills that contain user decision points. Any step labeled as a blocking point or user decision point must follow this protocol.

## Core Rules

- Decision points are blocking points. Pause and wait for an explicit user choice before continuing
- Use the current platform's available user input or confirmation mechanism to collect the choice; when platforms such as Claude Code provide `AskUserQuestion`, prefer `AskUserQuestion` to present single-select or multi-select options
- If the current tool list does not include `AskUserQuestion`, or the first `AskUserQuestion` call fails, treat structured questions as unavailable for this session; do not repeatedly retry `AskUserQuestion` for later decision points in the same session, and use the text-options fallback directly
- If the current platform has no structured question tool, ask clear options in the conversation and stop until the user replies
- Never substitute recommendation rules, defaults, historical preferences, or “the user would probably agree” for current confirmation
- Do not write state fields, execute the chosen branch, or auto-continue before the user explicitly chooses

## `AskUserQuestion` Priority Strategy

When using structured questions:

- Use a single-select `AskUserQuestion` question for single-choice decision points, and a multi-select question for multi-choice decision points
- Each option must include a short label and impact description; recommendations may explain tradeoffs but must not auto-select
- If the tool call succeeds, wait for the user to answer through that question; do not also print a duplicate text option list
- If the tool is missing, the call fails, or the host reports an error, record that structured questions are unavailable for this session, then use the text-options fallback

## Minimum Presentation Requirements

- State what the current decision point is deciding
- Present clear options; when the user must pick one option, keep the options mutually exclusive and actionable
- In text-options fallback mode, explicitly state "single-select" or "multi-select", number the options, describe each option's impact, ask the user to reply with the option number(s), then stop until the user replies
- Recommendations may explain tradeoffs, but may not replace user confirmation
- Only execute the corresponding commands or state updates after the user chooses
