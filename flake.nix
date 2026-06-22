{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    go-overlay.url = "github:purpleclay/go-overlay";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs =
    {
      self,
      nixpkgs,
      go-overlay,
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
      useLatestGoVersion = final: prev: {
        go_latest = final.go-bin.latestStable;
        # Use buildPackages so the Go toolchain runs on the build platform while
        # still cross-compiling for the target (matches nixpkgs' buildGo*Module),
        # otherwise cross builds pick the target-arch Go binary and fail to exec.
        buildGoLatestModule = prev.buildGoLatestModule.override { go = final.buildPackages.go-bin.latestStable; };
      };
      flake = flake-utils.lib.eachDefaultSystem (
        system:
        let
          pkgs = import nixpkgs {
            inherit system;
            config.allowUnfree = true;
            overlays = [
              self.overlays.default
              go-overlay.overlays.default
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
                go_latest
                govulncheck
                gofumpt
                golangci-lint
                govulncheck
                just
                mockgen
                nix-prefetch-docker
                pinact
                prek
                sd
                skopeo
              ];
              shellHook = ''
                prek install
              '';
            };

          formatter = pkgs.nixfmt-rfc-style;
        }
      );
    in
    flake // { inherit overlays; };
}
