package domain

// User エンティティと Username 値オブジェクト。
// username は 3〜20文字 [a-z0-9_]、小文字に正規化して保存する。
// パスワードの平文はここに置かない（ハッシュ化は infrastructure/auth）。
//
// TODO: User{ID, Username, CreatedAt} / ParseUsername / ValidatePassword
