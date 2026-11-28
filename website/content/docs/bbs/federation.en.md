---
title: Global Boards & Federation
weight: 40
---

gobot's Anonymous BBS supports **Global Boards** shared across multiple Discord servers, as well as **cross-instance Federation (Protocol v2)** connecting independent gobot deployments.

## Subscribing to Global Boards (`/bbs global`)

Server administrators (`Manage Server` or `bbs.admin`) can mount published Global Boards into their server channels:

- `/bbs global list` — List available Global Boards (both locally hosted and federated from external instances).
- `/bbs global subscribe` — Mount a Global Board to a channel in your server and start real-time synchronization.
- `/bbs global unsubscribe` — Unmount the Global Board and stop synchronization.

## How Federation Works (Origin / Listener)

- **Origin Instance**: Hosts the canonical Global Board data and streams new threads, posts, and moderation events via SSE (Server-Sent Events). The Origin also reconstructs SLIP context and enforces moderation rules consistently.
- **Listener Instance**: Subscribes to an Origin's Global Boards and mirrors them to its Discord servers in real time, with automatic resync after disconnects.
- **Federation SLIP Labels**: Origin peer configurations can attach instance-specific SLIP labels to federated posts.

## Posting to External Instances with OAuth 2.0

When a user on a Listener server posts or submits a report to a Global Board hosted by an external Origin, a one-time **OAuth 2.0 authorization** is required:

1. Clicking a post or report button presents an authorization link for the external instance.
2. Once authorized, the issued tokens are stored encrypted in the Listener database so subsequent posts work seamlessly.
3. Users can revoke federation tokens at any time via `/bbs policy`.

## Privacy and Disclaimer Notes

- Posts submitted to Global Boards are synchronized to all subscribing Discord servers and federated External Instances.
- For full details on external data synchronization and operator responsibility, see [BBS Terms of Service (Article 3)](/en/bbs/terms/) and [BBS Privacy Policy (Section 4)](/en/bbs/privacy/).
