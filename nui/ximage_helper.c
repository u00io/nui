//go:build linux
// +build linux

#include <X11/Xlib.h>
#include <X11/Xatom.h>
#include <X11/Xutil.h>
#include "ximage_helper.h"

void destroy_ximage(XImage* img) {
    XDestroyImage(img);
}

void maximizeWindow(Display* display, Window window) {
    Atom wmState = XInternAtom(display, "_NET_WM_STATE", False);
    Atom maxH = XInternAtom(display, "_NET_WM_STATE_MAXIMIZED_HORZ", False);
    Atom maxV = XInternAtom(display, "_NET_WM_STATE_MAXIMIZED_VERT", False);

    XClientMessageEvent xev = {0};
    xev.type = ClientMessage;
    xev.serial = 0;
    xev.send_event = True;
    xev.window = window;
    xev.message_type = wmState;
    xev.format = 32;
    xev.data.l[0] = 1; // _NET_WM_STATE_ADD
    xev.data.l[1] = maxH;
    xev.data.l[2] = maxV;
    xev.data.l[3] = 0;
    xev.data.l[4] = 0;

    Window root = DefaultRootWindow(display);
    XSendEvent(display, root, False, SubstructureRedirectMask | SubstructureNotifyMask, (XEvent*)&xev);
}

void restoreWindow(Display* display, Window window) {
    Atom wmState = XInternAtom(display, "_NET_WM_STATE", False);
    Atom maxH = XInternAtom(display, "_NET_WM_STATE_MAXIMIZED_HORZ", False);
    Atom maxV = XInternAtom(display, "_NET_WM_STATE_MAXIMIZED_VERT", False);

    XClientMessageEvent xev = {0};
    xev.type = ClientMessage;
    xev.serial = 0;
    xev.send_event = True;
    xev.window = window;
    xev.message_type = wmState;
    xev.format = 32;
    xev.data.l[0] = 0; // _NET_WM_STATE_REMOVE
    xev.data.l[1] = maxH;
    xev.data.l[2] = maxV;
    xev.data.l[3] = 0;
    xev.data.l[4] = 0;

    Window root = DefaultRootWindow(display);
    XSendEvent(display, root, False, SubstructureRedirectMask | SubstructureNotifyMask, (XEvent*)&xev);
}

void minimizeWindow(Display* display, Window window) {
    int screen = DefaultScreen(display);
    XIconifyWindow(display, window, screen);
}

// Marks `window` as a modal dialog owned by `parent`: sets WM_TRANSIENT_FOR so
// window managers group/stack it with its parent, tags it as a dialog via
// _NET_WM_WINDOW_TYPE, and requests _NET_WM_STATE_MODAL so compliant window
// managers (GNOME/KDE/XFCE) block input to the parent while it's open.
void setWindowModal(Display* display, Window window, Window parent) {
    XSetTransientForHint(display, window, parent);

    Atom wmWindowType = XInternAtom(display, "_NET_WM_WINDOW_TYPE", False);
    Atom wmWindowTypeDialog = XInternAtom(display, "_NET_WM_WINDOW_TYPE_DIALOG", False);
    XChangeProperty(display, window, wmWindowType, XA_ATOM, 32, PropModeReplace,
                    (unsigned char*)&wmWindowTypeDialog, 1);

    Atom wmState = XInternAtom(display, "_NET_WM_STATE", False);
    Atom stateModal = XInternAtom(display, "_NET_WM_STATE_MODAL", False);

    XClientMessageEvent xev = {0};
    xev.type = ClientMessage;
    xev.send_event = True;
    xev.window = window;
    xev.message_type = wmState;
    xev.format = 32;
    xev.data.l[0] = 1; // _NET_WM_STATE_ADD
    xev.data.l[1] = stateModal;
    xev.data.l[2] = 0;
    xev.data.l[3] = 0;
    xev.data.l[4] = 0;

    Window root = DefaultRootWindow(display);
    XSendEvent(display, root, False, SubstructureRedirectMask | SubstructureNotifyMask, (XEvent*)&xev);
}

