{ pkgs ? import <nixpkgs> {} }:

let libhandy = pkgs.libhandy.overrideAttrs(old: {
		name = "libhandy-1.0.1";
		src  = builtins.fetchGit {
			url = "https://gitlab.gnome.org/GNOME/libhandy.git";
			rev = "5cee0927b8b39dea1b2a62ec6d19169f73ba06c6";
		};
		patches = [];
	
		buildInputs = old.buildInputs ++ (with pkgs; [
			gnome3.librsvg
			gdk-pixbuf
		]);
	});

	mpv-mpris = pkgs.stdenv.mkDerivation {
		name = "mpv-mpris-0.5";

		src = pkgs.fetchFromGitHub {
			owner  = "hoyon";
			repo   = "mpv-mpris";
			rev    = "a95d2a5007b614b70f32981ea5b1dab90371a840";
			sha256 = "07p6li5z38pkfd40029ag2jqx917vyl3ng5p2i4v5a0af14slcnk";
		};

		dontFixup  = true;
		preInstall = "export HOME=$out";

		nativeBuildInputs = with pkgs; [
			mpv
			pkgconfig
			gnome3.glib
		];
	};

in pkgs.stdenv.mkDerivation rec {
	name = "cchat-gtk";
	version = "0.0.2";

	buildInputs = [ libhandy ] ++ (with pkgs; [
		gnome3.glib gnome3.gtk
		mpv
		ffmpeg
		fftw portaudio
	]);

	nativeBuildInputs = with pkgs; [
		pkgconfig go
	];

	# mpv MPRIS support.
	MPV_SCRIPTS = "${mpv-mpris}/.config/mpv/scripts/mpris.so";
}
