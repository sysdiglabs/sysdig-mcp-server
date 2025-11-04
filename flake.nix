{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
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
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };
      in
      {
        devShells.default =
          with pkgs;
          mkShell {
            packages = [
              go_1_25
              ginkgo
              mockgen
              gofumpt
              python3
              uv
              ruff
              basedpyright
              sysdig-cli-scanner
            ];
          };

        formatter = pkgs.nixfmt-rfc-style;
      }
    );
}
