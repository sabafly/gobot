---
title: Posting, Tripcodes & SLIP
weight: 20
---

Learn how threads, anonymous IDs, tripcodes, Watchoi (SLIP), and caps work in gobot's Anonymous BBS.

## Creating Threads and Posting Replies

1. Click a board or thread action button (**New Thread**, **Post Reply**, or **Reply**) to open the posting modal.
2. On first use, you will be prompted to review and accept the [BBS Terms of Service](/en/bbs/terms/), [Community Guidelines](/en/bbs/guidelines/), and [Privacy Policy](/en/bbs/privacy/).
3. Use the **Watch** button inside a thread to toggle DM notifications for new replies.

## Anonymous IDs and Tripcodes

- **Daily Board-Specific IDs**: Generated per board and per JST (Asia/Tokyo) calendar day using HMAC-SHA256 with a server secret. Other users cannot see your Discord username or User ID.
- **Tripcodes**: Append `#secret` to your display name to generate a verifiable tripcode.
- **Anchors & Media**: Reference earlier posts with `>>1` (up to 10 anchors per post) and attach media URLs (up to 10 per post).

## Watchoi (SLIP)

Depending on board settings or thread-level directives, posts can display a **Watchoi (SLIP)** tag derived from anonymous posting context:

| Mode | Display |
| --- | --- |
| `none` | No SLIP displayed. |
| `vvv` | Displays a playful nickname derived from non-identifying Discord traits (such as account age, badges, voice activity, and server size tier) and date/time. |
| `vvvvv` | Displays the `vvv` nickname plus an 8-character identifier (`abcd-efgh`, "Korokoro") combining a 4-character user-fixed prefix and a 4-character suffix incorporating the nickname/display name. |

Thread-level directives such as `!slip:vvv`, `!slip:vvvvv`, or `!extend` can also enable SLIP display per thread.

## Caps

Authorized users configured by the operator can select a **Cap** from the posting modal to post with a verified cap display name. On federated Global Boards, available caps are dynamically retrieved from the hosting Origin instance.
