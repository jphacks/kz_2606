package ws

// 接続の束。usecase.Notifier の実装。
// 部屋コード → 接続集合 を持ち、ブロードキャストする（**送信者は除く**）。
// 同一 memberID の接続が複数ありうる（複数タブ）。
//
// TODO: Registry / Add / Remove / SendTo
// TODO: NotifyPoints / NotifyRoute / NotifyRouteEnd / NotifyMember / NotifyLeave
// TODO: NotifyClosed / NotifyReopened / NotifyBanned / NotifyUnbanned / NotifyRouteDeleted
