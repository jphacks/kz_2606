# 開発環境。本体にはインストールせず、ここだけで Go / Node を揃える。
#
#   nix develop            シェルに入る
#   nix develop -c make    一発だけ実行する
#
# direnv を使うなら `direnv allow` で自動的に入る（.envrc 参照）。
#
# ⚠️ flake は git が追跡しているファイルしか見ない。
#    `git add flake.nix flake.lock .envrc` してから nix develop すること。
{
  description = "kz_2606 — 共同参加型GPSアート作成Webアプリ";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forEachSystem = f:
        nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      devShells = forEachSystem (pkgs: {
        default = pkgs.mkShell {
          name = "kz_2606";

          packages = with pkgs; [
            # --- backend ---
            go # go.mod の go 1.23 以上
            gopls # LSP
            gotools # goimports
            go-tools # staticcheck

            # --- frontend ---
            nodejs_22

            # --- 周辺 ---
            gnumake
            postgresql_16 # psql（DB本体は docker compose 側）
            cloudflared # トンネル
            jq
          ];

          shellHook = ''
            # NixOS では npm が落としてくる native addon（@rollup/rollup-linux-*,
            # @tailwindcss/oxide など）が libstdc++ を見つけられないことがある。
            export LD_LIBRARY_PATH="${pkgs.lib.makeLibraryPath [ pkgs.stdenv.cc.cc.lib ]}:$LD_LIBRARY_PATH"

            # go install したものをリポジトリ内に閉じる（ホームを汚さない）
            export GOPATH="$PWD/.direnv/go"
            export PATH="$GOPATH/bin:$PATH"

            echo "kz_2606 dev shell"
            echo "  go   $(go version | awk '{print $3}')"
            echo "  node $(node --version)"
            echo ""
            echo "  make tidy   go.sum を生成（初回必須）"
            echo "  make db     Postgres を起動"
            echo "  make server Go を :8080 で起動"
            echo "  make web    Vite を :5173 で起動"
          '';
        };
      });
    };
}
