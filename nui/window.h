#ifndef WINDOW_H
#define WINDOW_H


int InitWindow(void);
void ShowWindow(int windowId);
void ShowModalWindow(int windowId, int parentWindowId);
void RunEventLoop(void);

void CloseWindowById(int windowId);
void QuitApp(void);
void SetWindowTitle(int windowId, const char* title);
void SetWindowSize(int windowId, int width, int height);
void SetWindowPosition(int windowId, int x, int y);
void MinimizeWindow(int windowId);
void MaximizeWindow(int windowId);
void SetAppIconFromRGBA(const char* data, int width, int height);
void StartTimer(int windowId, double intervalMilliseconds);
void StopTimer(int windowId);
void UpdateWindow(int windowId);

void SetMacCursor(int cursorType);

int GetWindowPositionX(int windowId);
int GetWindowPositionY(int windowId);
int GetWindowWidth(int windowId);
int GetWindowHeight(int windowId);

int GetClientAreaWidth(int windowId);
int GetClientAreaHeight(int windowId);

int GetScreenWidth();
int GetScreenHeight();

// File dialogs. Each returns a malloc'd, UTF-8, NUL-terminated string that the
// caller must free() - NULL means the user cancelled. ShowOpenFileDialog joins
// multiple picked paths with '\n'. extensionsCSV is a comma-separated list of
// bare extensions (no leading '*' or '.', e.g. "txt,md"); NULL or "" allows
// any file. parentWindowId may be -1 for no owner.
char* ShowOpenFileDialog(int parentWindowId, const char* title, const char* defaultDirectory, const char* extensionsCSV, int allowMultiple);
char* ShowSaveFileDialog(int parentWindowId, const char* title, const char* defaultDirectory, const char* defaultFileName, const char* extensionsCSV);
char* ShowSelectDirectoryDialog(int parentWindowId, const char* title, const char* defaultDirectory);

// Shows/hides the titlebar's miniaturize (yellow) / zoom (green) buttons,
// e.g. for dialog-style windows that shouldn't offer them.
void SetWindowAllowMinimize(int windowId, int allow);
void SetWindowAllowMaximize(int windowId, int allow);

void go_on_paint(int hwnd, void* buffer, int width, int height);
void go_on_key_down(int hwnd, int keycode);
void go_on_key_up(int hwnd, int keycode);
void go_on_modifier_change(int hwnd, int shift, int ctrl, int alt, int cmd, int caps, int num, int fnKey);
void go_on_char(int hwnd, int codepoint);
void go_on_window_move(int hwnd, int x, int y);
void go_on_declare_draw_time(int hwnd, int time);

void go_on_mouse_down(int hwnd, int button, int x, int y);
void go_on_mouse_up(int hwnd, int button, int x, int y);
void go_on_mouse_move(int hwnd, int x, int y);
void go_on_mouse_scroll(int hwnd, float deltaX, float deltaY);
void go_on_mouse_enter(int hwnd);
void go_on_mouse_leave(int hwnd);
void go_on_mouse_double_click(int hwnd, int button, int x, int y);
void go_on_timer(int hwnd);
void go_on_resize(int windowId, int width, int height);
int go_on_close_request(int windowId);
void go_on_window_will_close(int windowId);

#endif
