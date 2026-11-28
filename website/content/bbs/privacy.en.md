---
title: gobot Anonymous BBS Privacy Policy
toc: true
---

The operator of gobot (hereinafter referred to as the "Operator") establishes this Privacy Policy (hereinafter referred to as the "Policy") regarding the handling of user information in the anonymous bulletin board feature (hereinafter referred to as the "Feature") of the Discord bot "gobot".

## 1. Information Collected and Retained

The Operator stores the following information in its database to provide, maintain, and prevent abuse of the Feature:

- **Consent Records**: Discord User ID, consented policy version, and timestamp of consent.
- **Board Data**: Post content, display name, tripcode, daily board-specific ID, Watchoi (SLIP), and post timestamp.
- **Notification Subscription Records**: Discord User ID of users subscribed to thread notifications, display locale, and registration timestamp.
- **Report Records**: Reporter's Discord User ID, target post/thread ID, reason for report, and handling status.
- **Audit Identification Information**: Poster's Discord User ID and an HMAC-SHA256 hash derived using a secret key (anonymous moderation hash).
- **Federation Authentication Tokens**: Encrypted OAuth 2.0 access and refresh tokens issued when a User authorizes posting or interacting with federated External Instances.

## 2. Purpose of Use

Collected information is used strictly within the scope of the following purposes:

1. Providing, maintaining, and improving the Feature, and delivering thread notifications (DM notifications).
2. Investigating violations of the Terms of Service and Community Guidelines, performing moderation, and preventing unauthorized access and trolling/spam.
3. Responding to user reports and enforcing access restrictions (Local Bans and Global Bans).
4. Responding to lawful inquiries, disclosure requests, and resolving disputes.

## 3. Protection of Anonymity and Non-Disclosure to Guild Administrators

1. Only daily, board-specific anonymous IDs are displayed on the boards; other users cannot view your Discord User ID or account name. However, on boards where Watchoi (SLIP) is enabled, or in threads with `!slip:vvv` / `!slip:vvvvv` / `!extend` specified where ID hiding is disabled, a playful nickname derived from anonymous Discord traits (such as account age, badges, voice activity, or general server tier/size without disclosing raw usernames or server names) and posting date/time will be publicly displayed. In addition, under `vvvvv` mode, an 8-character identifier in `abcd-efgh` format—combining a 4-character prefix fixed to your User ID and a 4-character suffix ("Korokoro") incorporating your display name—will also be publicly displayed.
2. Even when Guild Administrators use moderation features, raw Discord User IDs are never disclosed to them. Guild Administrators may only perform post masking/removal ("abone") and Local Bans on Guild-Local Boards using the anonymous moderation hash.

## 4. Operator's Use of Audit Information and Provision to Third Parties

1. Posters' Discord User IDs stored on the server are not viewed during normal operations.
2. The Operator will not provide user information to third parties except in the following cases:
   - **Synchronization to Global Boards and External Server Federation**: Posts submitted to boards under the Global category are synchronized and delivered in the same anonymous format to other Discord servers and federated External Instances (such as other gobot servers) subscribing to those boards. In addition, when a User authorizes OAuth 2.0 federation to post or submit reports on a Global Board hosted by an External Instance, User Content, report details, and Discord User ID / profile traits obtained via OAuth 2.0 are provided to that External Instance. The Operator assumes no responsibility whatsoever for how synchronizing servers or External Instances store, repost, delete, or manage such data.
   - **Disclosure Based on Laws and Regulations**: When requested pursuant to a court warrant, official law enforcement inquiry, lawful sender information disclosure request/order under the Information Distribution Platform Act of Japan (formerly the Provider Liability Limitation Act), or other applicable laws. The Operator shall not be liable for any damages incurred by a User due to disclosures made in good faith pursuant to legal requests, except in cases of willful misconduct or gross negligence by the Operator.
   - **Protection of Life and Property**: When urgently necessary to protect a person's life, body, or property, and obtaining the User's consent is difficult.
3. Solely for responding to reports, enforcing Global Bans against severe Terms violations or trolling, or complying with lawful disclosure requests, the Operator may inspect posters' Discord User IDs via audit logs.

## 5. External Links, Media Previews, and Discord Specifications

1. Posts containing media URLs are displayed via Discord's media preview mechanism. Accessing external links or using external services is subject to the respective third-party providers' privacy policies, and the Operator assumes no responsibility for information leaks, malware infections, or any other damages resulting therefrom.
2. Because the Feature operates on the Discord platform, data collection and processing by Discord Inc. are governed by Discord's Privacy Policy.

## 6. Security Measures, Retention Period, and Disclaimer

1. The Operator strives to prevent the leakage, loss, or damage of user information within a reasonable scope through measures such as database access controls, transport encryption (TLS / HTTPS), secret-key hashing (HMAC-SHA256), and encrypted storage of federation tokens. However, due to the inherent nature of internet communications and electronic data management, absolute security cannot be guaranteed. The Operator shall not be liable for any data breaches, leaks, or losses caused by unauthorized access, cyberattacks, software vulnerabilities, hardware failures, or other causes not attributable to the Operator.
2. Collected information is retained for the period necessary to provide the Feature, prevent abuse, and comply with legal obligations. However, the Operator does not guarantee permanent data retention and assumes no liability for data loss caused by system failures or specification changes.

## 7. Withdrawal of Consent and Data Deletion

1. Users may withdraw their consent to this Policy (and revoke federated OAuth 2.0 tokens) at any time via the `/bbs policy` command.
2. Upon withdrawal, consent records and active thread notification subscriptions (DM notification settings) are immediately deleted, and subsequent new postings, notification subscriptions, DM notifications, and reporting will be suspended.
3. Notwithstanding the preceding paragraph, existing anonymous post data itself is retained to preserve board integrity (problematic posts will be handled via "abone" through reporting and moderation). Furthermore, audit identification information and report records may continue to be retained as necessary to prevent recurring abuse, maintain active bans, and comply with lawful disclosure requests.

## 8. Amendments, Inquiries, and Prevailing Language

1. The Operator may amend this Policy in response to legal revisions or changes to the Feature's specifications.
2. For inquiries or requests regarding this Policy, please contact us at the Support Server.
3. In the event of any discrepancy between the Japanese version and this English translation, the Japanese version shall prevail.

Effective Date: November 26, 2026 (Last Updated: November 29, 2026)
