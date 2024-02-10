{
  description = "picsum.photos";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = inputs@{
    self,
    nixpkgs,
    flake-utils,
    ...
  }:
    flake-utils.lib.eachSystem [
      "x86_64-linux"
      "aarch64-linux"
      "aarch64-darwin"
    ] (system:
      let pkgs = nixpkgs.legacyPackages.${system}; in {
        packages = rec {
          default = everything;

          everything = pkgs.symlinkJoin {
            name = "picsum-photos-composite";
            paths = [
              picsum-photos
              image-service
            ];
          };

          picsum-photos = pkgs.buildGo122Module {
            name = "picsum-photos";
            src = ./.;
            CGO_ENABLED = 0;
            subPackages = ["cmd/picsum-photos"];
            doCheck = false; # Prevent make test from being ran
            vendorHash = (pkgs.lib.fileContents ./go.mod.sri);
            nativeBuildInputs = with pkgs; [
              tailwindcss
            ];
            preBuild = ''
              go generate ./...
            '';
          };

          image-service = pkgs.buildGo122Module {
            name = "image-service";
            src = ./.;
            subPackages = ["cmd/image-service"];
            doCheck = false; # Prevent make test from being ran
            vendorHash = (pkgs.lib.fileContents ./go.mod.sri);
            nativeBuildInputs = with pkgs; [
              pkg-config
            ];
            buildInputs = with pkgs; [
              vips
            ];
          };
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go_1_22
            gotools
            go-tools
            gopls
            delve
          ];
        };
      }
    ) // {
      nixosModules.default = { config, lib, pkgs, ... }:
        with lib;
        let cfg = config.picsum-photos.services.image-service;
        in {
          options.picsum-photos.services.image-service = {
          enable = mkEnableOption "Enable the image-service service";

          logLevel = mkOption {
            type = with types; enum [ "debug" "info" "warn" "error" "dpanic" "panic" "fatal" ];
            example = "debug";
            default = "info";
            description = "log level";
          };

          domain = mkOption {
            type = types.str;
            description = "Domain to listen to";
          };

          sockPath = mkOption rec {
            type = types.path;
            default = "/run/image-service/image-service.sock";
            example = default;
            description = "Unix domain socket to listen on";
          };

          environmentFile = mkOption {
            type = types.path;
            description = "Environment file";
          };

          storagePath = mkOption rec {
            type = types.path;
            default = "/var/lib/image-service";
            example = default;
            description = "Storage path";
          };
        };

        config = mkIf cfg.enable {
          users.groups.image-service = {};

          users.users.image-service = {
            createHome = true;
            isSystemUser = true;
            group = "image-service";
            home = "/var/lib/image-service";
          };

          systemd.services.image-service = {
            description = "image-service";
            wantedBy = [ "multi-user.target" ];

            script = ''
              exec ${self.packages.${pkgs.system}.image-service}/bin/image-service -log-level=${cfg.logLevel} -listen=${cfg.sockPath} -storage-path=${cfg.storagePath}
            '';

            serviceConfig = {
              EnvironmentFile = cfg.environmentFile;
              User = "image-service";
              Group = "image-service";
              Restart = "always";
              RestartSec = "30s";
              WorkingDirectory = "/var/lib/image-service";
              RuntimeDirectory = "image-service";
              RuntimeDirectoryMode = "0770";
              UMask = "007";
            };
          };

          services.nginx.virtualHosts."${cfg.domain}" = {
            locations."/" = {
              proxyPass = "http://unix:${cfg.sockPath}";
            };
          };
        };
      };
    };
}
