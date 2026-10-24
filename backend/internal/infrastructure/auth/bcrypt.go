package auth

// usecase.PasswordHasher の実装。golang.org/x/crypto/bcrypt、コストは既定(10)。
// 意図的に遅い（約70ms）ので、テストでは必ずダミーに差し替える。
//
// TODO: Hasher / Hash / Compare
