package usecase

// 登録とログイン。bcrypt も Cookie も知らない（port 経由）。
//
// 注意: ログイン失敗時に「存在しない」と「パスワードが違う」を区別しない。
//       区別するとユーザー名を列挙できてしまう。
// 仕様: DATA_MODEL.md 5章
//
// TODO: AuthService / Register / Login / Me
