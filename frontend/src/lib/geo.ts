// 距離計算と点のフィルタ。生のGPSはそのまま描くと団子とヒゲだらけになる。
// しきい値: 精度30m超 / 移動5m未満 / 速度14m/s超を捨てる。30秒以上空いたら線を切る。
//
// TODO: distance / pathLength / PointFilter / simplify / splitByGap

export {};
