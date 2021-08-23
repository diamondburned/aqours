{ pkgs ? import <unstable> {} }:

pkgs.stdenv.mkDerivation rec {
	name = "aqours";
	version = "0.0.2";

	CGO_ENABLED = "1";

	buildInputs = with pkgs; [
		gnome3.glib gnome3.gtk
		mpv
		libhandy
		ffmpeg
		fftw portaudio
	];

	nativeBuildInputs = with pkgs; [
		pkgconfig go
	];
}
