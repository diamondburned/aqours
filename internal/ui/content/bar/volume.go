package bar

import (
	"log"
	"runtime/debug"

	"github.com/diamondburned/aqours/internal/ui/content/bar/controls"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/gotk3/gotk3/gtk"
)

type Volume struct {
	gtk.Box

	Icon   *gtk.Image
	Mute   *gtk.ToggleButton
	Slider *gtk.Scale

	volume float64
	muted  bool
}

var volumeSliderCSS = css.PrepareClass("volume-slider", `
	scale {
		margin: 0;
		padding-left: 2px;
	}
`)

var muteButtonCSS = css.PrepareClass("mute-button", `
	button {
		margin:  0;
		color:   @theme_fg_color;
		opacity: 0.5;
		box-shadow: none;
		background: none;
	}

	button:hover {
		opacity: 1;
	}
`)

func NewVolume(parent ParentController) *Volume {
	icon, _ := gtk.ImageNew()
	icon.Show()

	mute, _ := gtk.ToggleButtonNew()
	mute.SetRelief(gtk.RELIEF_NONE)
	mute.SetImage(icon)
	mute.Show()
	muteButtonCSS(mute)

	slider, _ := gtk.ScaleNewWithRange(gtk.ORIENTATION_HORIZONTAL, 0, 100, 1)
	slider.SetSizeRequest(100, -1)
	slider.SetDrawValue(false)
	slider.Show()
	controls.CleanScaleCSS(slider)
	volumeSliderCSS(slider)

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(mute, false, false, 0)
	box.PackStart(slider, true, true, 0)
	box.SetVAlign(gtk.ALIGN_CENTER)
	box.SetHAlign(gtk.ALIGN_END)
	box.SetHExpand(true)

	volume := &Volume{
		Box:    *box,
		Icon:   icon,
		Mute:   mute,
		Slider: slider,
		volume: 100,
		muted:  false,
	}

	mute.SetActive(volume.muted)
	slider.SetValue(volume.volume)
	volume.updateIcon()

	mute.Connect("toggled", func() {
		volume.muted = mute.GetActive()
		volume.updateIcon()
		slider.SetSensitive(!volume.muted) // no sense to change volume while muted
		parent.SetMute(volume.muted)
	})

	slider.Connect("value-changed", func() {
		volume.volume = clampVolume(slider.GetValue())
		volume.updateIcon()
		parent.SetVolume(volume.volume)
	})

	return volume
}

// SetVolume sets the volume and triggers the callback to parent.
func (v *Volume) SetVolume(perc float64) {
	log.Println("(*Volume).SetVolume called:", string(debug.Stack()))
	v.Slider.SetValue(perc)
}

// GetVolume returns the volume.
func (v *Volume) GetVolume() float64 {
	return v.volume
}

// IsMuted returns true if the volume is muted.
func (v *Volume) IsMuted() bool {
	return v.muted
}

func (v *Volume) updateIcon() {
	var icon string

	switch {
	case v.volume < 1 || v.muted:
		icon = "audio-volume-muted-symbolic"
	case v.volume < 30:
		icon = "audio-volume-low-symbolic"
	case v.volume < 80:
		icon = "audio-volume-medium-symbolic"
	default:
		icon = "audio-volume-high-symbolic"
	}

	v.Icon.SetFromIconName(icon, gtk.ICON_SIZE_BUTTON)
}

func clampVolume(perc float64) float64 {
	switch {
	case perc < 0:
		return 0
	case perc > 100:
		return 100
	default:
		return perc
	}
}
