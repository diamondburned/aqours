package bar

import (
	"fmt"

	"github.com/diamondburned/aqours/internal/ui/content/bar/controls"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type VisualizerStatus int8

const (
	VisualizerDisabled VisualizerStatus = iota - 1
	VisualizerOnlyPlaying
	VisualizerAlwaysOn
	VisualizerMuted
	visualizerStatusLen
)

// Visualizer code is removed. Add it back once catnip is ported to GTK4 and
// isn't garbage.
const defaultVisStatus = VisualizerDisabled

func (vs VisualizerStatus) IsPaused(paused bool) bool {
	switch vs {
	case VisualizerMuted, VisualizerDisabled:
		return true
	case VisualizerAlwaysOn:
		return false
	case VisualizerOnlyPlaying:
		fallthrough
	default:
		return paused
	}
}

func (vs VisualizerStatus) String() string {
	switch vs {
	case VisualizerDisabled:
		return "Disabled"
	case VisualizerMuted:
		return "Muted"
	case VisualizerOnlyPlaying:
		return "Only when Playing"
	case VisualizerAlwaysOn:
		return "Always On"
	default:
		return fmt.Sprintf("VisualizerStatus(%d)", vs)
	}
}

func (vs VisualizerStatus) cycle() VisualizerStatus {
	return (vs + 1) % visualizerStatusLen
}

func (vs VisualizerStatus) icon() string {
	switch vs {
	case VisualizerDisabled:
		return ""
	case VisualizerMuted:
		return "microphone-sensitivity-muted-symbolic"
	case VisualizerAlwaysOn:
		return "microphone-sensitivity-high-symbolic"
	case VisualizerOnlyPlaying:
		fallthrough
	default:
		return "microphone-sensitivity-medium-symbolic"
	}
}

type VolumeController interface {
	ParentController
	SetVisualize(visualize VisualizerStatus)
}

type Volume struct {
	gtk.Box

	VisIcon   *gtk.Image
	Visualize *gtk.Button

	Icon   *gtk.Image
	Mute   *gtk.ToggleButton
	Slider *gtk.Scale

	volume    float64
	muted     bool
	visualize VisualizerStatus
}

var volumeSliderCSS = css.PrepareClass("volume-slider", `
	.volume-slider {
		margin: 0;
		padding-left: 2px;
	}
`)

var muteButtonCSS = css.PrepareClass("mute-button", ``)

var rightButtonCSS = css.PrepareClass("right-button", `
	.right-button {
		margin:  0;
		color:   @theme_fg_color;
		opacity: 0.5;
		box-shadow: none;
		background: none;
	}
	.right-button:hover {
		opacity: 1;
	}
`)

var volumeCSS = css.PrepareClass("volume", "")

func NewVolume(parent VolumeController) *Volume {
	visIcon := gtk.NewImage()

	visualize := gtk.NewButton()
	visualize.SetChild(visIcon)
	rightButtonCSS(visualize)

	icon := gtk.NewImage()

	mute := gtk.NewToggleButton()
	mute.SetChild(icon)

	muteButtonCSS(mute)
	rightButtonCSS(mute)

	slider := gtk.NewScaleWithRange(gtk.OrientationHorizontal, 0, 100, 1)
	slider.SetSizeRequest(100, -1)
	slider.SetDrawValue(false)
	slider.SetHExpand(true)

	controls.CleanScaleCSS(slider)
	volumeSliderCSS(slider)

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	if defaultVisStatus != VisualizerDisabled {
		box.Append(visualize)
	}

	box.Append(mute)
	box.Append(slider)
	box.SetVAlign(gtk.AlignCenter)
	box.SetHAlign(gtk.AlignEnd)
	box.SetHExpand(true)

	volume := &Volume{
		Box:       *box,
		VisIcon:   visIcon,
		Visualize: visualize,
		Icon:      icon,
		Mute:      mute,
		Slider:    slider,
		volume:    100,
		muted:     false,
		visualize: defaultVisStatus,
	}
	volumeCSS(volume)

	mute.SetActive(volume.muted)
	slider.SetValue(volume.volume)
	volume.updateIcon()

	visualize.Connect("clicked", func() {
		volume.visualize = volume.visualize.cycle()
		volume.updateIcon()
		parent.SetVisualize(volume.visualize)
	})

	mute.Connect("toggled", func() {
		volume.muted = mute.Active()
		volume.updateIcon()
		slider.SetSensitive(!volume.muted) // no sense to change volume while muted
		parent.SetMute(volume.muted)
	})

	slider.Connect("value-changed", func() {
		volume.volume = clampVolume(slider.Value())
		volume.updateIcon()
		parent.SetVolume(volume.volume)
	})

	return volume
}

// SetVolume sets the volume and triggers the callback to parent.
func (v *Volume) SetVolume(perc float64) {
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

// VisualizerStatus returns the internal visualizer status.
func (v *Volume) VisualizerStatus() VisualizerStatus {
	return v.visualize
}

func (v *Volume) updateIcon() {
	v.updateVisualizeIcon()
	v.updateVolumeIcon()
}

func (v *Volume) updateVisualizeIcon() {
	v.VisIcon.SetFromIconName(v.visualize.icon())
	v.Visualize.SetTooltipText(v.visualize.String())
}

func (v *Volume) updateVolumeIcon() {
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

	v.Icon.SetFromIconName(icon)
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
