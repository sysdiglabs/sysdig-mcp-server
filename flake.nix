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
    let
      overlays.default = final: prev: {
        sysdig-mcp-server =
          if prev.stdenv.isLinux then
            prev.pkgsStatic.callPackage ./package.nix { }
          else
            prev.callPackage ./package.nix { };
      };
      flake = flake-utils.lib.eachDefaultSystem (
        system:
        let
          pkgs = import nixpkgs {
            inherit system;
            config.allowUnfree = true;
            overlays = [ self.overlays.default ];
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
                go
                govulncheck
                gofumpt
                golangci-lint
                just
                mockgen
                nix-prefetch-docker
                pre-commit
                skopeo
                sd
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
