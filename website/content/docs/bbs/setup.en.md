---
title: Setup & Board Management
weight: 10
---

This guide explains how server administrators enable the Anonymous BBS feature and create local categories and boards.

## Required Permissions

Management subcommands (`/bbs setup`, `/bbs category`, `/bbs board`, `/bbs global`) can be executed by members who have either:

- Discord **`Manage Server`** permission (unless `bbs.admin` is explicitly denied), or
- The gobot permission node **`bbs.admin`**

## 1. Initial Guild Setup (`/bbs setup`)

Run `/bbs setup` first to configure server-level BBS settings, such as optional moderation log channels.

## 2. Managing Categories (`/bbs category`)

Organize your Guild-Local Boards into categories:

- `/bbs category create` — Create a new local category.
- `/bbs category list` — List categories in the current server.
- `/bbs category delete` — Delete an unused category.

## 3. Creating and Editing Local Boards (`/bbs board`)

Create a Guild-Local Board under a category and mount it to a Discord forum or text channel:

- `/bbs board create` — Configure board name, category, target channel, default anonymous name, and Watchoi (SLIP) mode (`none`, `vvv`, or `vvvvv`).
- `/bbs board list` — List local boards in the server.
- `/bbs board edit` — Update an existing board's name, default anonymous name, or SLIP mode.
- `/bbs board delete` — Delete a local board.

## 4. Forum Channels vs. Text Channels

- **Forum Channels**: Each BBS thread is created as its own Discord forum post, with a pinned board guide post for creating threads and browsing the thread list.
- **Text Channels**: A board control panel provides buttons to browse threads, create new threads, and read/reply through the interactive reader UI.
