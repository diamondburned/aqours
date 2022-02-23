package css

import (
	"bytes"
	"log"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var globalCSS bytes.Buffer

// PrepareClass prepares the CSS and returns a function that applies the class
// onto the given widget. The CSS should be for the class only, and the class
// should reflect what's in the CSS block.
func PrepareClass(class, css string) (attach func(gtk.Widgetter)) {
	globalCSS.WriteString(css)

	return func(widget gtk.Widgetter) {
		w := gtk.BaseWidget(widget)
		w.AddCSSClass(class)
	}
}

// Prepare parses the given CSS and returns the CSSProvider.
func Prepare(css string) *gtk.CSSProvider {
	p := gtk.NewCSSProvider()
	p.ConnectParsingError(func(sec *gtk.CSSSection, err error) {
		// Optional line parsing routine.
		loc := sec.StartLocation()
		lines := strings.Split(css, "\n")
		log.Printf("CSS error (%v) at line: %q", err, lines[loc.Lines()])
	})
	p.LoadFromData(css)
	return p
}

// AddGlobal adds CSS to the global CSS buffer.
func AddGlobal(css string) {
	globalCSS.WriteString(css)
}

// LoadGlobal loads the global CSS buffer into the given display.
func LoadGlobal(disp *gdk.Display) {
	prov := Prepare(globalCSS.String())
	gtk.StyleContextAddProviderForDisplay(disp, prov, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
