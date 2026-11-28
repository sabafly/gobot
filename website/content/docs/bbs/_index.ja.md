---
title: 匿名掲示板 (BBS) ガイド
weight: 10
---

gobot の匿名掲示板（BBS）機能は、Discord のフォーラムチャンネルやテキストチャンネル上に 5ch 風の匿名スレッド・レス環境を構築できる機能です。

ギルド内限定の「ギルドローカル板」と、複数サーバーや他インスタンス間で共有される「グローバル板」の両方に対応しています。

{{< cards >}}
  {{< card link="setup" title="導入と板の作成" icon="cog" subtitle="/bbs setup・カテゴリ管理・板の作成とマウント手順" >}}
  {{< card link="posting" title="投稿機能・トリップ・SLIP" icon="pencil-alt" subtitle="スレ立て・レス・アンカー・トリップ・ワッチョイ (SLIP)・キャップ" >}}
  {{< card link="moderation" title="モデレーションと同意管理" icon="shield-check" subtitle="通報・あぼーん・ローカル/グローバルアク禁・dat落ち・/bbs policy" >}}
  {{< card link="federation" title="グローバル板とフェデレーション" icon="globe-alt" subtitle="Origin / Listener 構成・グローバル板購読・OAuth 2.0 連携" >}}
{{< /cards >}}

## 利用前に確認すること

- 匿名掲示板機能は**オプトイン方式**です。利用前に [BBS 利用規約](/bbs/terms/)・[BBS コミュニティガイドライン](/bbs/guidelines/)・[BBS プライバシーポリシー](/bbs/privacy/) への同意が必要です。
- 各コマンドの詳細なオプションは [コマンド一覧: `/bbs`](/docs/commands/bbs/) を参照してください。
