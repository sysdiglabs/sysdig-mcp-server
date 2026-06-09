{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    nixpkgs-25-11.url = "github:NixOS/nixpkgs/nixos-25.11";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs =
    {
      self,
      nixpkgs,
      nixpkgs-25-11,
      flake-utils,
    }:
    let
      overlays.default = final: prev: {
        sysdig-mcp-server =
          if prev.stdenv.isLinux then
            prev.pkgsStatic.callPackage ./package.nix { }
          else
            prev.callPackage ./package.nix { };
      };
      useLatestGoVersion =
        final: prev:
        let
          nixpkgs = import nixpkgs-25-11 { inherit (prev) system; };
        in
        {
          go_1_26 = nixpkgs.go_1_26;
        };
      flake = flake-utils.lib.eachDefaultSystem (
        system:
        let
          pkgs = import nixpkgs {
            inherit system;
            config.allowUnfree = true;
            overlays = [
              self.overlays.default
              useLatestGoVersion
            ];
          };
        in
        {
          packages = {
            inherit (pkgs) sysdig-mcp-server;
            default = pkgs.sysdig-mcp-server;
            sysdig-mcp-server-image-amd64 = pkgs.pkgsCross.gnu64.callPackage ./docker.nix { };
            sysdig-mcp-server-image-aarch64 = pkgs.pkgsCross.aarch64-multiplatform.callPackage ./docker.nix { };
          };
          devShells.default =
            with pkgs;
            mkShell {
              packages = [
                ginkgo
                go_1_26
                gofumpt
                golangci-lint
                govulncheck
                just
                mockgen
                nix-prefetch-docker
                pinact
                pre-commit
                sd
                skopeo
              ];
              shellHook = ''
                pre-commit install
              '';
            };

          formatter = pkgs.nixfmt-rfc-style;
        }
      );
    in
    flake // { inherit overlays; };
}
