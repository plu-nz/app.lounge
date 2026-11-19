#ifndef LOUNGE_RENDER_H
#define LOUNGE_RENDER_H

#include <mpv/client.h>

// lounge_attach moves the window's webview into a GtkOverlay above a
// GtkGLArea that libmpv renders into. Call it on the GTK main thread.
int lounge_attach(void *gtk_window, mpv_handle *handle);

// lounge_detach frees the render context before the mpv core is destroyed.
// Call it on the GTK main thread.
void lounge_detach(void);

// lounge_node_json renders an mpv node as JSON. The caller frees the result.
char *lounge_node_json(const mpv_node *node);

#endif
