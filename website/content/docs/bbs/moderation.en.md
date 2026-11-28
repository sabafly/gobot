---
title: Moderation & Consent
weight: 30
---

This page explains privacy-preserving moderation tools and how users manage their own BBS consent and federation links.

## Guild-Local Moderation

Guild Administrators (`Manage Server` or `bbs.admin`) manage moderation for Guild-Local Boards:

- **No Raw User ID Exposure**: Raw Discord User IDs and usernames are never disclosed to Guild Administrators during moderation.
- **Abone (Masking/Removal)**: Mask or hide rule-breaking posts and threads.
- **Archive Thread**: Lock and archive a thread to stop further replies.
- **Local Ban**: Ban a user from Guild-Local Boards using their anonymous moderation hash.

## Reporting and Global Moderation

- Users can submit a **Report** with a reason from any post or thread action menu.
- Reports on Global Boards and severe policy violations are reviewed by the Operator (or the hosting External Instance operator), who may apply abone or **Global Bans**.

## Automatic Archiving (Dat-Ochi)

A background scheduler automatically archives threads that reach the reply limit or remain inactive beyond the configured retention period.

## Managing Consent (`/bbs policy`)

Any user can run `/bbs policy` at any time to:

- View the current [BBS Terms of Service](/en/bbs/terms/), [Community Guidelines](/en/bbs/guidelines/), and [Privacy Policy](/en/bbs/privacy/)
- Check their consent status and consented version
- **Withdraw consent** (immediately deleting consent records and DM notification subscriptions, and suspending further posting or reporting)
- **Revoke OAuth 2.0 tokens** linked to federated External Instances
