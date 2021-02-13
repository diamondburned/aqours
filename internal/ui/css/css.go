package css

import (
	"log"
	"runtime/debug"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

type StyleContexter interface {
	GetStyleContext() (*gtk.StyleContext, error)
}

func PrepareClass(class, css string) (attach func(StyleContexter)) {
	prov := Prepare(css)

	return func(ctx StyleContexter) {
		s, _ := ctx.GetStyleContext()
		s.AddProvider(prov, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
		s.AddClass(class)
	}
}

func Prepare(css string) *gtk.CssProvider {
	p, _ := gtk.CssProviderNew()
	if err := p.LoadFromData(css); err != nil {
		log.Fatalf("CSS fail (%v) at %s\n", err, debug.Stack())
	}
	return p
}

// StyleContext gets the style context from the given contexter. Nil is
// returned on any error.
func StyleContext(ctx StyleContexter) *gtk.StyleContext {
	v, _ := ctx.GetStyleContext()
	return v
}

var cssRepos = map[string]*gtk.CssProvider{}

func getDefaultScreen() *gdk.Screen {
	d, _ := gdk.DisplayGetDefault()
	s, _ := d.GetDefaultScreen()
	return s
}

func loadProviders(screen *gdk.Screen) {
	for file, repo := range cssRepos {
		gtk.AddProviderForScreen(
			screen, repo,
			uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION),
		)
		// mark as done
		delete(cssRepos, file)
	}
}

func LoadGlobal(name, css string) {
	prov, _ := gtk.CssProviderNew()
	if err := prov.LoadFromData(css); err != nil {
		log.Fatalf("Failed to parse CSS in %s: %v\n", name, err)
		return
	}

	cssRepos[name] = prov
}
