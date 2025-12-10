# にじ通

### 概要
にじさんじライバーのYoutube生放送、動画を事前に通知をする非公式Discordサーバーです。  
- 「〇〇っていうゲームの生配信みたいけど、見逃したくないし事前通知してほしい！」  
- 「いろんなライバーさんの歌枠をリアルタイムで聴きたい！」

など思っている人にはおすすめのサーバーです。  

![image](https://github.com/user-attachments/assets/b65e49c1-2ba0-4da1-aff4-f674fcbf6a94)

### 使用技術
- [Go](https://go.dev/)
- [MySQL](https://www.mysql.com/jp/)
- [sqlx](https://github.com/jmoiron/sqlx)
- [Google Cloud](https://cloud.google.com/?hl=ja)

以下は WEB版のみで使用している技術
- [FCM](https://firebase.google.com/docs/cloud-messaging?hl=ja)
- [Astro](https://astro.build/)
- [Bulma](https://bulma.io/)

### WEB版にじ通
開発停止していますがWEB版も公開しています。歌ってみた動画を通知する機能のみ使用できます。※将来廃止予定
- https://niji-tuu.app

### ディレクトリ構成（それぞれの機能の説明）
- bot
  - Discordボット関連
  - 管理者用のコマンド処理やユーザの通知登録の処理を行う

- new_video
  - 新着動画関連
  - 新着動画があるか定期的にチェックする
  - あった場合はイベントを送信する

- song
  - 歌動画関連
  - new_videoから受け取ったイベントから動画情報を取得し、歌動画かチェックする
  - 歌動画の場合は指定時刻に通知するタスクを作成する
  - 通知は プッシュ通知、Xのポスト、Discordサーバーでの通知の三種類を実施

- keyword
  - キーワード関連
  - new_videoから受け取ったイベントから動画情報を取得し、特定のキーワードが含まれているかチェック
  - 含まれている場合は指定時刻に通知するタスクを作成する
  - 通知はDiscordサーバーのみで実施

- internal
  - 上記の機能で共通して使用する共通機能

- frontend
  - WEBページ関連
  - プッシュ通知を登録する

- testdata
  - それぞれの機能のテストで使用するデータ

### インフラ
![にじ通_インフラ](https://github.com/user-attachments/assets/2fa8bb44-40f5-4b9f-a83c-011a51e238f7)
