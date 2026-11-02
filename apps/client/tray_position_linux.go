//go:build linux && !android && !server

package main

/*
#cgo pkg-config: x11
#cgo !gtk3 pkg-config: gtk4
#cgo gtk3 pkg-config: gtk+-3.0
#include <gtk/gtk.h>
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#ifdef GDK_WINDOWING_X11
#include <gdk/x11/gdkx.h>
#endif

static void prepare_tray_window(GtkWindow *window, int x, int y) {
    if (!window) return;
#if GTK_MAJOR_VERSION == 3
    gtk_window_set_skip_taskbar_hint(window, TRUE);
    gtk_window_move(window, x, y);
#elif defined(GDK_WINDOWING_X11)
    // Create the native surface without mapping it. The window manager can
    // then honor the requested position and taskbar hint on the first map.
    gtk_widget_realize(GTK_WIDGET(window));
    GdkSurface *surface = gtk_native_get_surface(GTK_NATIVE(window));
    if (!surface || !GDK_IS_X11_SURFACE(surface)) return;
    gdk_x11_surface_set_skip_taskbar_hint(surface, TRUE);
    Display *display = gdk_x11_display_get_xdisplay(gdk_surface_get_display(surface));
    Window xid = gdk_x11_surface_get_xid(surface);
    XSizeHints hints = {0};
    long supplied;
    XGetWMNormalHints(display, xid, &hints, &supplied);
    hints.flags |= USPosition;
    hints.x = x;
    hints.y = y;
    XSetWMNormalHints(display, xid, &hints);
    XMoveWindow(display, xid, x, y);
    XFlush(display);
#endif
}

static int tray_pointer(int *x, int *y) {
    Display *display = XOpenDisplay(NULL);
    if (!display) return 0;
    Window root, child;
    int wx, wy;
    unsigned int mask;
    int found = 0;
    for (int i = 0; i < ScreenCount(display); i++) {
        if (XQueryPointer(display, RootWindow(display, i), &root, &child,
                          x, y, &wx, &wy, &mask)) { found = 1; break; }
    }
    XCloseDisplay(display);
    return found;
}
*/
import "C"

import (
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func configureTrayPosition(app *application.App, tray *application.SystemTray, window *application.WebviewWindow) {
	// X11 panels do not all provide usable StatusNotifier Activate coordinates.
	// Capture the actual pointer at the icon click instead of trusting (0, 0).
	// Keep Wails' compositor handling on native Wayland.
	if os.Getenv("XDG_SESSION_TYPE") == "wayland" || os.Getenv("DISPLAY") == "" {
		return
	}
	tray.OnClick(func() {
		var px, py C.int
		if C.tray_pointer(&px, &py) == 0 {
			tray.ToggleWindow()
			return
		}
		application.InvokeAsync(func() {
			if window.IsVisible() {
				window.Hide()
				return
			}
			for _, screen := range app.Screen.GetAll() {
				bounds := screen.PhysicalBounds
				if int(px) < bounds.X || int(px) >= bounds.X+bounds.Width || int(py) < bounds.Y || int(py) >= bounds.Y+bounds.Height {
					continue
				}
				scale := float64(screen.ScaleFactor)
				if scale <= 0 {
					scale = 1
				}
				x := screen.Bounds.X + int(float64(int(px)-bounds.X)/scale)
				y := screen.Bounds.Y + int(float64(int(py)-bounds.Y)/scale)
				work := screen.WorkArea
				if work.Width <= 0 || work.Height <= 0 {
					work = screen.Bounds
				}
				width, height := window.Width(), window.Height()
				left, top := x-width/2, y+8
				if y > screen.Bounds.Y+screen.Bounds.Height/2 {
					top = y - height - 8
				}
				left = max(work.X, min(left, work.X+work.Width-width))
				top = max(work.Y, min(top, work.Y+work.Height-height))
				// Position the native surface and set its taskbar hint before showing it.
				C.prepare_tray_window((*C.GtkWindow)(window.NativeWindow()), C.int(bounds.X+int(float64(left-screen.Bounds.X)*scale)), C.int(bounds.Y+int(float64(top-screen.Bounds.Y)*scale)))
				window.Show()
				window.Focus()
				return
			}
			tray.ShowWindow()
		})
	})
}
