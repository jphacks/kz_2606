package persistence

// usecase.UserRepository の Postgres 実装。
// username は小文字で保存し、UNIQUE 違反は domain.ErrUsernameTaken に変換する。
//
// TODO: UserRepository / Create / FindByUsername / FindByID
