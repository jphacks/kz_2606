package rest

// 認証ハンドラ。
//   POST /api/auth/register  登録 → そのままログイン（Cookieを返す）
//   POST /api/auth/login
//   POST /api/auth/logout
//   GET  /api/auth/me        ← フロント起動時に必ず叩く。無いとリロードでログイン状態が消える
// 仕様: DATA_MODEL.md 5章
//
// TODO: authHandler / register / login / logout / me
