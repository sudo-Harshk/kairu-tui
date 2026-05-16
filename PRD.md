# Product Requirements Document (PRD): Kairu TUI Enhancements

**Date:** Sunday, 17 May 2026
**Role:** Product Head
**Status:** Revised - Focus on Remaining Debt

## 1. Executive Summary
Following a technical audit, it has been confirmed that the core cross-platform performance bugs (Windows notification lag and shell portability) have been resolved. This revised PRD focuses on the remaining gaps in user feedback, error handling, and feature discoverability for Kairu TUI.

## 2. Remaining Problem Statement
1.  **Notification Transparency:** While the outbox retry logic works, users have no visibility into the `notification_outbox.json` status or the ability to manually flush/clear failed notifications.
2.  **Soundscape Setup Friction:** If the external `SoundscapePlayer` (e.g., `mpv`) is missing or misconfigured, the app provides a generic error rather than actionable setup instructions.
3.  **Documentation Gap:** Powerful features such as Soundscapes, Streak Recovery, and the Template Manager remain undocumented in the user-facing guides, leading to low discovery.

## 3. Outstanding Requirements

### 3.1 Notification Dashboard (Settings UI)
*   **Objective:** Provide transparency for the background notification engine.
*   **Requirements:**
    *   Add a "Queue Status" line to the Settings screen showing the number of pending retries in the outbox.
    *   Implement an "Action" in the Backup/Tools section of Settings to "Clear Notification Queue."

### 3.2 Enhanced Audio Error Handling
*   **Objective:** Reduce friction for users setting up soundscapes.
*   **Requirements:**
    *   Improve the error message when `SoundscapePlayer` fails to start. Instead of "Failed to start," show: "Audio player not found. Ensure 'mpv' is installed or update 'soundscape_player' in kairu.yaml."

### 3.3 Documentation Sync
*   **Objective:** Formally announce and guide users through the new productivity features.
*   **Requirements:**
    *   **README.md:** Add a "Soundscapes" section explaining how to add audio files and use `Ctrl+M`.
    *   **docs/usage.md:** Add a "Streak Recovery" section explaining the 1-day grace period logic.
    *   **docs/usage.md:** Document the `Ctrl+P` Template Manager workflow.

## 4. Technical Constraints
*   **TUI Consistency:** All changes must remain keyboard-first and compatible with `charmbracelet/bubbletea`.
*   **Local-First:** No new external dependencies for data storage; continue using JSON/YAML.

## 5. Engineering Timeline (Remaining)
1.  **Phase 1 (Error Handling):** Actionable soundscape error messages.
2.  **Phase 2 (Settings UI):** Outbox status and "Clear Queue" action.
3.  **Phase 3 (Documentation):** Complete the README and feature docs.

