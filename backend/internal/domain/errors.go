package domain

// ドメインエラー。上の層は errors.Is で判別して HTTP ステータスに変換する。
//
// TODO: ErrRoomNotFound / ErrInvalidRoomCode / ErrMemberNotFound
