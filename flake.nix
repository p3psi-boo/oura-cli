{
  description = "oura-cli development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        formatter = pkgs.nixfmt-rfc-style;

        devShells.default = pkgs.mkShell {
          # Keep this minimal and Go-focused; add tools as needed.
          packages = with pkgs; [
            go
            gopls
            golangci-lint
            gotestsum
            delve
            gofumpt
            git
          ];

          shellHook = ''
            echo "Loaded oura-cli dev shell"
            command -v go >/dev/null && go version || true
          '';
        };
      }
    );
}
