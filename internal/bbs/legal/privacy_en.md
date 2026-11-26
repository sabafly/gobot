# gobot Anonymous BBS Privacy Policy

1. Data We Process and Store
To operate the BBS and prevent abuse, gobot stores the following records:
- Consent Records: Your Discord User ID, accepted policy version, and timestamp.
- Board Content: Thread titles, reply text, display names, tripcodes, daily per-board anonymous IDs, and SLIP identifiers.
- Thread Watch Subscriptions: Subscribed user Discord User ID, locale preference, and creation timestamp (deleted upon consent withdrawal).
- Abuse Reports: Reporter Discord User ID, target post/thread ID, report reason, and resolution status.
- Audit Metadata: Author Discord User ID and HMAC-SHA256 moderation hash (`PosterHash`).

2. Anonymity and Non-Disclosure to Server Administrators
- By default, only daily per-board anonymous IDs are shown publicly in posts, hiding your Discord User ID and username from other users. On boards or threads where Watchoi (`SLIP`) is enabled (`SLIPVVVVV` or `!slip:vvvvv` / `!extend` with ID display enabled), an HMAC-derived `SLIP` identifier (`abcd-efgh`) derived from your Discord User ID is also displayed publicly and stays consistent across dates and boards.
- Server administrators cannot view your raw Discord User ID through any BBS moderation command. Local moderation on server-owned local boards operates strictly on opaque local `PosterHash` identifiers.

3. Operator Audit Access
- Stored Discord User IDs are not visible during normal operation.
- Only designated gobot operators may inspect audit metadata strictly for handling abuse reports, enforcing global bans against severe violations, or complying with valid legal requests.

4. Cross-Server Distribution (Global Boards)
- Posts made on Global boards are mirrored in anonymous form to all Discord servers subscribed to that board.
- External media URLs included in posts are rendered via Discord's MediaGallery proxy; external hosts are governed by their own privacy policies.

5. Withdrawal of Consent
- You may withdraw your consent at any time via `/bbs policy`. Withdrawal deletes your consent record and active thread watch subscriptions, preventing further posting, viewing, watch subscriptions, and DM notifications while preserving existing anonymous thread structure (problematic posts can be soft-deleted via "Abone").
