# Misskey TUI Client Requirements Definition

This document outlines the features and specifications for a new Misskey TUI client.

## Phase 1: Core Functionality (Minimum Viable Product)

The absolute essential features for the client to be minimally usable.

### 1. Authentication
- On first launch, the user should be able to log in by providing their Misskey instance URL and access token.

### 2. Timeline Display
- Ability to switch between and view Home (HTL), Local (LTL), Social (STL), and Global (GTL) timelines.
- New notes should stream in real-time.

### 3. Posting Notes
- Create and send a new note.
- Reply to a selected note.
- Renote a selected note.

### 4. Reactions
- Ability to react to a selected note with a predefined, simple emoji (e.g., 👍, ❤️).

### 5. Notifications
- View a list of notifications, such as replies and reactions.

---

## Phase 2: Enhanced Features

Features to build upon the core functionality for a richer user experience.

### 6. Advanced Display
- Render basic MFM (Misskey Flavored Markdown) formatting (e.g., bold, links).
- Display custom emoji (text representation like `:emoji:` is acceptable).
- Render URLs for images and videos as clickable links that open in an external browser.

### 7. User Interaction
- View user profiles (name, ID, bio).
- Follow/unfollow users.

### 8. Advanced Posting
- Create a quote renote.
- Select reaction emoji from a picker-like UI.

### 9. Search
- Search for notes by keyword.
- Search for users.

---

## Phase 3: Advanced & Customization Features

Advanced features for power users and Misskey-specific functionality.

### 10. Misskey-Specific Features
- View posts from Antennas.
- View and post messages in Channels.

### 11. Customization
- Allow users to customize keyboard shortcuts.
- Allow users to change the color theme.

### 12. Multi-Account Support
- Ability to switch between multiple Misskey accounts.

---

## Screen Layout Proposal

Two primary directions for the TUI layout are proposed.

### A. Single View
- A simple layout where the entire screen is dedicated to a single view (e.g., a timeline). The user switches between content (different timelines, notifications, post composition window) via keyboard shortcuts.
- **Pros:** Relatively simple to implement, clean and uncluttered view.
- **Cons:** Cannot view multiple information streams simultaneously.

### B. Multi-Column View (TweetDeck-style)
- The screen is divided into several columns, each displaying a different information stream (e.g., Home, Notifications, and Local timelines side-by-side).
- **Pros:** High information density, allowing users to monitor multiple streams at once.
- **Cons:** More complex to implement, can feel crowded on smaller terminal screens.

---

## Initial Recommendation

The recommended starting point is to implement the **Phase 1 (MVP)** features using a **Single View** layout. This provides a solid foundation to build upon.
