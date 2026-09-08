//go:build linux
// +build linux

#pragma once
#include <X11/Xlib.h>

void destroy_ximage(XImage* img);
void maximizeWindow(Display* display, Window window);
void restoreWindow(Display* display, Window window);
void minimizeWindow(Display* display, Window window);
void setWindowModal(Display* display, Window window, Window parent);
void setWindowDecorations(Display* display, Window window, int allowMinimize, int allowMaximize);
