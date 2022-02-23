{ pkgs ? import <nixpkgs> {} }:

let go = pkgs.go.overrideAttrs (old: {
		version = "1.17.6";
		src = builtins.fetchurl {
			url    = "https://go.dev/dl/go1.17.6.src.tar.gz";
			sha256 = "sha256:1j288zwnws3p2iv7r938c89706hmi1nmwd8r5gzw3w31zzrvphad";
		};
		doCheck = false;
		patches = [
			# cmd/go/internal/work: concurrent ccompile routines
			(builtins.fetchurl "https://github.com/diamondburned/go/commit/4e07fa9fe4e905d89c725baed404ae43e03eb08e.patch")
			# cmd/cgo: concurrent file generation
			(builtins.fetchurl "https://github.com/diamondburned/go/commit/432db23601eeb941cf2ae3a539a62e6f7c11ed06.patch")
		];
	});

in pkgs.stdenv.mkDerivation rec {
	name = "aqours";
	version = "0.0.2";

	CGO_ENABLED = "1";

	buildInputs = with pkgs; [
		gobject-introspection
		gnome3.glib
		(gnome3.gtk or gtk3)
		gtk4
		mpv
		ffmpeg
		# fftw portaudio
	];

	nativeBuildInputs = [ go pkgs.pkgconfig ];
}
