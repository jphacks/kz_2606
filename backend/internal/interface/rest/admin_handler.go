package rest

// owner 専用の操作。状態を変えたら Notifier で全員に通知する。
//   POST   /api/rooms/{code}/close
//   POST   /api/rooms/{code}/reopen
//   POST   /api/rooms/{code}/members/{memberId}/ban
//   DELETE /api/rooms/{code}/members/{memberId}/ban
//   DELETE /api/rooms/{code}/routes/{routeId}
// 仕様: DATA_MODEL.md 5章・7章
//
// TODO: adminHandler / close / reopen / ban / unban / deleteRoute
